package v2ray

import (
	"context"
	"net"
	"reflect"
	"strconv"
	"time"

	"github.com/devfeel/mapper"
	"github.com/gin-gonic/gin"
	"github.com/v2fly/v2ray-core/v5/app/observatory"
	pb "github.com/v2fly/v2ray-core/v5/app/observatory/command"
	"github.com/v2rayA/v2rayA/db/configure"
	"github.com/v2rayA/v2rayA/pkg/util/log"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var (
	ApiProducts = []string{
		"observatory",
		"running_state",
	}
	ApiFeed *Feed
)

const (
	ApiFeedBoxSize  = 10
	ApiFeedInterval = 1 * time.Second
)

type OutboundStatus struct {
	Alive bool  `json:"alive"`
	Delay int64 `json:"delay"`
	//LastErrorReason string           `json:"last_error_reason"`
	OutboundTag  string           `json:"outbound_tag"`
	Which        *configure.Which `json:"which"`
	LastSeenTime int64            `json:"last_seen_time"`
	LastTryTime  int64            `json:"last_try_time"`
}

func init() {
	mapper.Register(&observatory.OutboundStatus{})

	ApiFeed = NewSubscriptions(ApiFeedBoxSize)
	for _, product := range ApiProducts {
		ApiFeed.RegisterProduct(product)
	}
}

type ObservatoryResp struct {
	OutboundName string
	Resp         *pb.GetOutboundStatusResponse
}

func extractOutboundStatuses(resp *pb.GetOutboundStatusResponse) []*observatory.OutboundStatus {
	if resp == nil {
		return nil
	}
	// v2fly-compatible shape
	if s := resp.GetStatus(); s != nil {
		if os := s.GetStatus(); len(os) > 0 {
			return os
		}
	}
	// xray/v2ray-compat variants may expose different field/method names.
	rv := reflect.ValueOf(resp)
	if rv.Kind() != reflect.Ptr || rv.IsNil() {
		return nil
	}
	candidates := []string{"GetOutboundStatus", "GetOutboundStatuses", "GetResult", "GetStats"}
	for _, method := range candidates {
		mv := rv.MethodByName(method)
		if !mv.IsValid() || mv.Type().NumIn() != 0 || mv.Type().NumOut() != 1 {
			continue
		}
		out := mv.Call(nil)[0]
		if out.Kind() == reflect.Slice {
			result := make([]*observatory.OutboundStatus, 0, out.Len())
			for i := 0; i < out.Len(); i++ {
				if v, ok := out.Index(i).Interface().(*observatory.OutboundStatus); ok {
					result = append(result, v)
				}
			}
			if len(result) > 0 {
				return result
			}
		}
	}
	return nil
}

func getObservatoryResponses(conn *grpc.ClientConn, observatoryTags []string) (r []ObservatoryResp, err error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if len(observatoryTags) == 0 {
		observatoryTags = append(observatoryTags, "")
	}
	methods := []string{
		"/v2ray.core.app.observatory.command.ObservatoryService/GetOutboundStatus",
		"/xray.core.app.observatory.command.ObservatoryService/GetOutboundStatus",
	}
	for _, tag := range observatoryTags {
		req := &pb.GetOutboundStatusRequest{Tag: tag}
		var resp *pb.GetOutboundStatusResponse
		var callErr error
		for _, method := range methods {
			resp = new(pb.GetOutboundStatusResponse)
			callErr = conn.Invoke(ctx, method, req, resp)
			if callErr == nil {
				r = append(r, ObservatoryResp{OutboundName: tag, Resp: resp})
				break
			}
			if status.Code(callErr) != codes.Unimplemented && status.Code(callErr) != codes.Unknown {
				return nil, callErr
			}
		}
		if callErr != nil {
			return nil, callErr
		}
	}
	return r, nil
}

// ObservatoryProducer monitors outbound status via gRPC API and publishes to ApiFeed.
// This function is only compatible with v2ray-core v5+ API structure.
// For xray-core, this function should not be called as it uses different gRPC services.
func ObservatoryProducer(apiPort int, observatoryTags []string) (closeFunc func()) {
	closed := make(chan struct{})
	go func() {
		const product = "observatory"
		var conn *grpc.ClientConn
	nextLoop:
		for {
			select {
			case <-closed:
				return
			default:
			}
			p := ProcessManager.Process()
			if p == nil {
				time.Sleep(ApiFeedInterval)
				continue
			}
			// Set up a connection to the server.
			if conn == nil {
				ctx, cancel := context.WithTimeout(context.Background(), ApiFeedInterval)
				defer cancel()
				c, err := grpc.DialContext(
					ctx,
					net.JoinHostPort("127.0.0.1", strconv.Itoa(apiPort)),
					grpc.WithInsecure(),
					grpc.WithBlock(),
				)
				if err != nil {
					log.Warn("ObservatoryProducer: did not connect: %v", err)
					continue nextLoop
				}
				defer c.Close()
				conn = c
			}
			resps, err := getObservatoryResponses(conn, observatoryTags)
			if err != nil {
				if status.Code(err) == codes.Unavailable {
					// the connection is reliable, and reconnect
					conn = nil
					continue nextLoop
				}
				log.Warn("ObservatoryProducer: %v", err)
			} else {
				css := configure.GetConnectedServers()
				for _, r := range resps {
					outboundStatus := extractOutboundStatuses(r.Resp)
					os := make([]OutboundStatus, len(outboundStatus))
					for i := range outboundStatus {
						_ = mapper.AutoMapper(outboundStatus[i], &os[i])
						index := p.tag2WhichIndex[os[i].OutboundTag]
						if index < 0 || index >= css.Len() {
							continue
						}
						os[i].Which = css.Get()[index]
					}
					msg := gin.H{
						"outboundName":   r.OutboundName,
						"outboundStatus": os,
					}
					ApiFeed.ProductMessage(product, msg)
				}
			}
			time.Sleep(ApiFeedInterval)
		}
	}()
	return func() {
		close(closed)
	}
}
