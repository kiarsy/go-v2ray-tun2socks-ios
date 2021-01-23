package tun2socks

import (
	"context"
	"runtime/debug"
	"strings"
	"time"

	"github.com/eycorsican/go-tun2socks/common/log"
	"github.com/eycorsican/go-tun2socks/core"
	"github.com/eycorsican/go-tun2socks/proxy/socks"
	"github.com/eycorsican/go-tun2socks/proxy/v2ray"

	vcore "github.com/v2ray/v2ray-core"
	vproxyman "github.com/v2ray/v2ray-core/app/proxyman"
)

type PacketFlow interface {
	WritePacket(packet []byte)
}

func InputPacket(data []byte) {
	lwipStack.Write(data)
}

var lwipStack core.LWIPStack

func StartSocks(packetFlow PacketFlow, proxyHost string, proxyPort int) {
	if packetFlow != nil {
		lwipStack = core.NewLWIPStack()
		core.RegisterTCPConnHandler(socks.NewTCPHandler(proxyHost, uint16(proxyPort)))
		core.RegisterUDPConnHandler(socks.NewUDPHandler(proxyHost, uint16(proxyPort), 30*time.Second))
		core.RegisterOutputFn(func(data []byte) (int, error) {
			packetFlow.WritePacket(data)
			return len(data), nil
		})
	}
}

func StartV2Ray(packetFlow PacketFlow, configBytes []byte, proxyHost string, proxyPort int) {
	if packetFlow == nil {
		return
	}

	lwipStack = core.NewLWIPStack()
	// v, err := vcore.StartInstance("json", configBytes)
	v, err := vcore.StartInstance("json", configBytes)
	if err != nil {
		log.Fatalf("start V instance failed: %v", err)
	}

	sniffingConfig := &vproxyman.SniffingConfig{
		Enabled:             true,
		DestinationOverride: strings.Split("tls,http", ","),
	}

	debug.SetGCPercent(5)
	ctx := vproxyman.ContextWithSniffingConfig(context.Background(), sniffingConfig)
	// vproxyman.ContextWithSniffingConfig(context.Background(), sniffingConfig)

	// Register tun2socks connection handlers.
	vhandler := v2ray.NewHandler(ctx, v)
	core.RegisterTCPConnectionHandler(vhandler)
	core.RegisterUDPConnectionHandler(vhandler)

	// Write IP packets back to TUN.
	core.RegisterOutputFn(func(data []byte) (int, error) {
		if !isStopped {
			packetFlow.WritePacket(data)
		}
		return len(data), nil
	})

	// if packetFlow != nil {
	// 	lwipStack = core.NewLWIPStack()
	// 	core.RegisterTCPConnHandler(socks.NewTCPHandler(proxyHost, uint16(proxyPort)))
	// 	core.RegisterUDPConnHandler(socks.NewUDPHandler(proxyHost, uint16(proxyPort), 30*time.Second))
	// 	core.RegisterOutputFn(func(data []byte) (int, error) {
	// 		packetFlow.WritePacket(data)
	// 		return len(data), nil
	// 	})
	// }

	// core.RegisterTCPConnHandler(v2ray.NewTCPHandler(ctx, v))

	// core.RegisterUDPConnHandler(v2ray.NewUDPHandler(ctx, v, 30*time.Second))
	// core.RegisterOutputFn(func(data []byte) (int, error) {
	// 	packetFlow.WritePacket(data)
	// 	runtime.GC()
	// 	debug.FreeOSMemory()
	// 	return len(data), nil
	// })
}
