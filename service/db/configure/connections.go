package configure

import (
	"database/sql"
	"fmt"

	"github.com/v2rayA/v2rayA/db"
)

type dbServerRef struct {
	id       int64
	typ      TouchType
	id1      int
	subIndex int
}

func ensureOutboundTx(tx *sql.Tx, outbound string) error {
	_, err := tx.Exec("INSERT OR IGNORE INTO outbound_names (name, sort) VALUES (?, COALESCE((SELECT MAX(sort) + 1 FROM outbound_names), 0))", outbound)
	return err
}

func resolveWhichServerIDTx(tx *sql.Tx, wt Which) (int64, error) {
	if wt.ID <= 0 {
		return 0, fmt.Errorf("invalid server id: %d", wt.ID)
	}
	var serverID int64
	switch wt.TYPE {
	case ServerType:
		err := tx.QueryRow("SELECT id FROM servers WHERE type = 'server' AND sort = ?", wt.ID-1).Scan(&serverID)
		if err == sql.ErrNoRows {
			return 0, fmt.Errorf("server id is out of range: %d", wt.ID)
		}
		return serverID, err
	case SubscriptionServerType:
		if wt.Sub < 0 {
			return 0, fmt.Errorf("invalid subscription index: %d", wt.Sub)
		}
		var subID int64
		err := tx.QueryRow("SELECT id FROM subscriptions WHERE sort = ?", wt.Sub).Scan(&subID)
		if err == sql.ErrNoRows {
			return 0, fmt.Errorf("subscription index is out of range: %d", wt.Sub)
		}
		if err != nil {
			return 0, err
		}
		err = tx.QueryRow("SELECT id FROM servers WHERE type = 'subscription_server' AND sub_id = ? AND sort = ?", subID, wt.ID-1).Scan(&serverID)
		if err == sql.ErrNoRows {
			return 0, fmt.Errorf("subscription server is out of range: sub=%d id=%d", wt.Sub, wt.ID)
		}
		return serverID, err
	default:
		return 0, fmt.Errorf("invalid touch type: %q", wt.TYPE)
	}
}

func getConnectedServersByOutbound(outbound string) (*Whiches, error) {
	var whiches *Whiches
	err := db.WithBusyRetry(func() error {
		rows, err := db.GetDB().Query(`
			SELECT oc.server_id, s.type, s.sort, COALESCE(sub.sort, -1)
			FROM outbound_connections oc
			JOIN servers s ON s.id = oc.server_id
			LEFT JOIN subscriptions sub ON sub.id = s.sub_id
			WHERE oc.outbound_name = ?
			ORDER BY oc.sort, oc.id`, outbound)
		if err != nil {
			return err
		}
		defer rows.Close()

		next := &Whiches{}
		for rows.Next() {
			var serverID int64
			var typ string
			var sort, subSort int
			if err := rows.Scan(&serverID, &typ, &sort, &subSort); err != nil {
				return err
			}
			wt := &Which{ID: sort + 1, Outbound: outbound}
			switch typ {
			case "server":
				wt.TYPE = ServerType
			case "subscription_server":
				if subSort < 0 {
					// Stale row; a repair/prune pass will remove it.
					continue
				}
				wt.TYPE = SubscriptionServerType
				wt.Sub = subSort
			default:
				continue
			}
			next.Touches = append(next.Touches, wt)
		}
		if err := rows.Err(); err != nil {
			return err
		}
		whiches = next
		return nil
	})
	if err != nil {
		return nil, err
	}
	if whiches.Len() == 0 {
		legacy, migrated := migrateLegacyConnectedServers(outbound)
		if migrated {
			return legacy, nil
		}
	}
	return whiches, nil
}

func migrateLegacyConnectedServers(outbound string) (*Whiches, bool) {
	bucket := fmt.Sprintf("outbound.%v", outbound)
	legacy := &Whiches{}
	if err := db.Get(bucket, "connectedServers", legacy); err != nil || legacy.Len() == 0 {
		return nil, false
	}
	valid := make([]Which, 0, legacy.Len())
	for _, wt := range legacy.Get() {
		if wt == nil {
			continue
		}
		copy := *wt
		copy.Outbound = outbound
		if !isWhichInRange(copy) {
			continue
		}
		valid = append(valid, copy)
	}
	if len(valid) == 0 {
		_ = db.Delete(bucket, "connectedServers")
		return nil, false
	}
	if err := replaceConnects(outbound, valid); err != nil {
		return nil, false
	}
	_ = db.Delete(bucket, "connectedServers")
	whiches, err := getConnectedServersByOutbound(outbound)
	if err != nil {
		return nil, false
	}
	return whiches, true
}

func isWhichInRange(wt Which) bool {
	if wt.ID <= 0 {
		return false
	}
	switch wt.TYPE {
	case ServerType:
		return wt.ID <= GetLenServers()
	case SubscriptionServerType:
		return wt.Sub >= 0 && wt.Sub < GetLenSubscriptions() && wt.ID <= GetLenSubscriptionServers(wt.Sub)
	default:
		return false
	}
}

func clearConnects(outbound string) error {
	return db.ReadModifyWrite(func(tx *sql.Tx) error {
		if err := ensureOutboundTx(tx, outbound); err != nil {
			return err
		}
		_, err := tx.Exec("DELETE FROM outbound_connections WHERE outbound_name = ?", outbound)
		return err
	})
}

func addConnect(wt Which) error {
	return db.ReadModifyWrite(func(tx *sql.Tx) error {
		if wt.Outbound == "" {
			wt.Outbound = "proxy"
		}
		if err := ensureOutboundTx(tx, wt.Outbound); err != nil {
			return err
		}
		serverID, err := resolveWhichServerIDTx(tx, wt)
		if err != nil {
			return err
		}
		var nextSort int
		if err := tx.QueryRow("SELECT COALESCE(MAX(sort) + 1, 0) FROM outbound_connections WHERE outbound_name = ?", wt.Outbound).Scan(&nextSort); err != nil {
			return err
		}
		_, err = tx.Exec("INSERT OR IGNORE INTO outbound_connections (outbound_name, server_id, sort) VALUES (?, ?, ?)", wt.Outbound, serverID, nextSort)
		return err
	})
}

func replaceConnects(outbound string, touches []Which) error {
	return db.ReadModifyWrite(func(tx *sql.Tx) error {
		if err := ensureOutboundTx(tx, outbound); err != nil {
			return err
		}
		serverIDs := make([]int64, 0, len(touches))
		seen := make(map[int64]struct{})
		for _, wt := range touches {
			wt.Outbound = outbound
			serverID, err := resolveWhichServerIDTx(tx, wt)
			if err != nil {
				return err
			}
			if _, ok := seen[serverID]; ok {
				continue
			}
			seen[serverID] = struct{}{}
			serverIDs = append(serverIDs, serverID)
		}
		if _, err := tx.Exec("DELETE FROM outbound_connections WHERE outbound_name = ?", outbound); err != nil {
			return err
		}
		for i, serverID := range serverIDs {
			if _, err := tx.Exec("INSERT INTO outbound_connections (outbound_name, server_id, sort) VALUES (?, ?, ?)", outbound, serverID, i); err != nil {
				return err
			}
		}
		return nil
	})
}

func removeConnect(wt Which) error {
	return db.ReadModifyWrite(func(tx *sql.Tx) error {
		if wt.Outbound == "" {
			wt.Outbound = "proxy"
		}
		serverID, err := resolveWhichServerIDTx(tx, wt)
		if err != nil {
			return err
		}
		result, err := tx.Exec("DELETE FROM outbound_connections WHERE outbound_name = ? AND server_id = ?", wt.Outbound, serverID)
		if err != nil {
			return err
		}
		rows, _ := result.RowsAffected()
		if rows == 0 {
			return fmt.Errorf("given server cannot be found in database")
		}
		_, err = tx.Exec(`
			UPDATE outbound_connections
			SET sort = (
				SELECT COUNT(*) FROM outbound_connections oc2
				WHERE oc2.outbound_name = outbound_connections.outbound_name AND oc2.sort < outbound_connections.sort
			)
			WHERE outbound_name = ?`, wt.Outbound)
		return err
	})
}
