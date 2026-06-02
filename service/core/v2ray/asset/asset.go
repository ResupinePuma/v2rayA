package asset

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"time"

	"github.com/adrg/xdg"
	"github.com/muhammadmuzzammil1998/jsonc"
	"github.com/v2rayA/v2rayA/common/files"
	"github.com/v2rayA/v2rayA/conf"
	"github.com/v2rayA/v2rayA/pkg/util/log"
)

func GetV2rayLocationAssetOverride() string {
	if assetDir := conf.GetEnvironmentConfig().V2rayAssetsDirectory; assetDir != "" {
		return assetDir
	}
	if assetDir := os.Getenv("V2RAY_LOCATION_ASSET"); assetDir != "" {
		return assetDir
	}
	if assetDir := os.Getenv("XRAY_LOCATION_ASSET"); assetDir != "" {
		return assetDir
	}
	// Keep geoip.dat/geosite.dat next to v2raya.db by default. This makes the
	// asset directory stable across restarts and matches the directory that users
	// usually mount/persist together with the SQLite database.
	return conf.GetEnvironmentConfig().Config
}

func GetV2rayLocationAsset(filename string) (string, error) {
	const folder = "v2raya"

	assetDir := GetV2rayLocationAssetOverride()
	if assetDir == "" {
		assetDir = conf.GetEnvironmentConfig().Config
	}
	if err := os.MkdirAll(assetDir, 0755); err != nil {
		return "", err
	}
	target := filepath.Join(assetDir, filename)
	if _, err := os.Stat(target); err == nil {
		return target, nil
	} else if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return "", err
	}

	// If the user explicitly configured an asset directory, keep the old lookup
	// behavior and download missing assets into that directory.
	if conf.GetEnvironmentConfig().V2rayAssetsDirectory != "" || os.Getenv("V2RAY_LOCATION_ASSET") != "" || os.Getenv("XRAY_LOCATION_ASSET") != "" {
		if runtime.GOOS != "windows" {
			for _, searchPath := range []string{
				filepath.Join("/usr/local/share", folder, filename),
				filepath.Join("/usr/share", folder, filename),
			} {
				if _, err := os.Stat(searchPath); err == nil {
					return searchPath, nil
				} else if err != nil && !errors.Is(err, fs.ErrNotExist) {
					return "", err
				}
			}
		}
		return target, nil
	}

	// Best-effort migration from the previous XDG data location: copy the asset
	// next to v2raya.db, then use the stable config directory path from now on.
	if runtime.GOOS != "windows" {
		if oldPath, err := xdg.SearchDataFile(filepath.Join(folder, filename)); err == nil {
			if copyErr := copyFile(oldPath, target); copyErr == nil {
				return target, nil
			} else {
				log.Warn("failed to migrate %s from %s to %s: %v", filename, oldPath, target, copyErr)
			}
		}
	}
	return target, nil
}

func copyFile(from, to string) error {
	in, err := os.Open(from)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.OpenFile(to, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		_ = out.Close()
		return err
	}
	return out.Close()
}

func DoesV2rayAssetExist(filename string) bool {
	fullpath, err := GetV2rayLocationAsset(filename)
	if err != nil {
		return false
	}
	_, err = os.Stat(fullpath)
	if err != nil {
		return false
	}
	return true
}

func GetGFWListModTime() (time.Time, error) {
	fullpath, err := GetV2rayLocationAsset("LoyalsoldierSite.dat")
	if err != nil {
		return time.Now(), err
	}
	return files.GetFileModTime(fullpath)
}

func GetConfigBytes() (b []byte, err error) {
	b, err = os.ReadFile(GetV2rayConfigPath())
	if err != nil {
		log.Warn("failed to get config: %v", err)
		return
	}
	b = jsonc.ToJSON(b)
	return
}

func GetV2rayConfigPath() (p string) {
	return path.Join(conf.GetEnvironmentConfig().Config, "config.json")
}

func GetV2rayConfigDirPath() (p string) {
	return conf.GetEnvironmentConfig().V2rayConfigDirectory
}

func GetNftablesConfigPath() (p string) {
	return path.Join(conf.GetEnvironmentConfig().Config, "v2raya.nft")
}

func Download(url string, to string) (err error) {
	log.Info("Downloading %v to %v", url, to)
	c := http.Client{Timeout: 90 * time.Second}
	resp, err := c.Get(url)
	if err != nil || resp.StatusCode != 200 {
		if err == nil {
			defer resp.Body.Close()
			err = fmt.Errorf("code: %v %v", resp.StatusCode, resp.Status)
		}
		return err
	}
	defer resp.Body.Close()
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	return os.WriteFile(to, b, 0644)
}
