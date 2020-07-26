package core

import (
	"fmt"
	log "github.com/sjqzhang/seelog"
	"github.com/xukgo/gfs/constDefine"
	"net"
	"os"
	"strings"
)

func (this *Server) GetClusterNotPermitMessage(clientIp string) string { //r *http.Request
	var (
		message string
	)
	message = fmt.Sprintf(constDefine.CONST_MESSAGE_CLUSTER_IP, clientIp)
	return message
}
func (this *Server) IsPeer(clientIP string) bool {
	var (
		peer  string
		bflag bool
		cidr  *net.IPNet
		err   error
	)
	IsPublicIP := func(IP net.IP) bool {
		if IP.IsLoopback() || IP.IsLinkLocalMulticast() || IP.IsLinkLocalUnicast() {
			return false
		}
		if ip4 := IP.To4(); ip4 != nil {
			switch true {
			case ip4[0] == 10:
				return false
			case ip4[0] == 172 && ip4[1] >= 16 && ip4[1] <= 31:
				return false
			case ip4[0] == 192 && ip4[1] == 168:
				return false
			default:
				return true
			}
		}
		return false
	}
	//return true
	if this.util.Contains("0.0.0.0", this.confRepo.GetAdminIps()) {
		if IsPublicIP(net.ParseIP(clientIP)) {
			return false
		}
		return true
	}
	if this.util.Contains(clientIP, this.confRepo.GetAdminIps()) {
		return true
	}
	for _, v := range this.confRepo.GetAdminIps() {
		if strings.Contains(v, "/") {
			if _, cidr, err = net.ParseCIDR(v); err != nil {
				log.Error(err)
				return false
			}
			if cidr.Contains(net.ParseIP(clientIP)) {
				return true
			}
		}
	}
	realIp := os.Getenv("GFS_IP")
	if realIp == "" {
		realIp = this.util.GetPulicIP()
	}
	if clientIP == "127.0.0.1" || clientIP == realIp {
		return true
	}
	clientIP = "http://" + clientIP
	bflag = false
	for _, peer = range this.confRepo.GetPeers() {
		if strings.HasPrefix(peer, clientIP) {
			bflag = true
			break
		}
	}
	return bflag
}
