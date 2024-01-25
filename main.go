package main

import (
	"encoding/json"
	"fmt"
	"gopkg.in/yaml.v3"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"os/exec"
)

// const serverUrl = "http://metadata-server.kubevirt.svc.cluster.local:80"
const serverUrl = "http://192.168.3.233:8080"

type RootPasswdInfo struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type CommandsConfig struct {
	Commands []string `json:"commands" yaml:"commands"`
}

type ScaleRootfs struct {
	ScaleRootfs CommandsConfig `json:"scale_rootfs" yaml:"scale_rootfs"`
}

func main() {

	yamlFile, err := ioutil.ReadFile("/etc/virtinit/config.yaml")
	if err != nil {
		fmt.Println("Error reading YAML file:", err)
		panic(err)
	}

	if !ChangeInstanceHostname() {
		log.Printf("set hostname failed")
	}

	log.Printf("set hostname completed")

	if !ChangeInstanceRootPassword() {
		log.Printf("set root Password failed")
	}

	log.Printf("set root Password completed")

	if !ChangeInstanceRootfsSize(yamlFile) {
		log.Printf("auto set root fs failed")
	}
	log.Printf("auto set root fs completed")

	log.Printf("virtual machine Initialization completed")
}

func ChangeInstanceHostname() bool {
	hostnameUrl := serverUrl + "/kubevirt/latest/meta-data/hostname"
	request := InitializeHTTPClient(hostnameUrl)
	httpClient := http.Client{}
	resp, err := httpClient.Do(request)
	if err != nil {
		log.Printf("get hostname failed : %s\n", err)
		return false
	}

	rawResp, _ := ioutil.ReadAll(resp.Body)
	if resp.StatusCode == http.StatusForbidden {
		resp.StatusCode = http.StatusInternalServerError
		return false
	}

	command := fmt.Sprintf("hostnamectl set-hostname %s", string(rawResp))
	err = exec.Command("bash", "-c", command).Run()
	if err != nil {
		log.Printf("set hostname failed : %s\n", err)
		return false
	}
	return true
}

func ChangeInstanceRootfsSize(configByte []byte) bool {

	//config.yaml获取command []string
	var scaleRootfs ScaleRootfs
	err := yaml.Unmarshal(configByte, &scaleRootfs)
	if err != nil {
		log.Printf("unmarshal config.yaml is failed, because: %s\n", err)
		return false
	}

	err = exec.Command("bash", "-c", SliceString2BashString(scaleRootfs.ScaleRootfs.Commands)).Run()
	if err != nil {
		log.Printf(err.Error())
		log.Printf("	resize rootfs failed : %s\n", err)
		return false
	}
	log.Printf("	resize rootfs success")
	return true
}

func ChangeInstanceRootPassword() bool {
	hostnameUrl := serverUrl + "/kubevirt/latest/user-data"
	request := InitializeHTTPClient(hostnameUrl)
	httpClient := http.Client{}
	resp, err := httpClient.Do(request)
	if err != nil {
		log.Printf("get root password failed : %s\n", err)
		return false
	}

	rawResp, _ := ioutil.ReadAll(resp.Body)
	if resp.StatusCode == http.StatusForbidden {
		resp.StatusCode = http.StatusInternalServerError
		return false
	}

	var passInfo RootPasswdInfo
	if err = json.Unmarshal(rawResp, &passInfo); err != nil {
		log.Println("json Unmarshal failed: " + string(rawResp))
		return false
	}

	command := fmt.Sprintf("echo '%s' | passwd --stdin %s", passInfo.Password, passInfo.Username)
	fmt.Println(command)
	err = exec.Command("bash", "-c", command).Run()
	if err != nil {
		log.Printf("set root Password failed : %s\n", err)
		return false
	}
	return true
}

func InitializeHTTPClient(url string) *http.Request {
	request, err := http.NewRequest("GET", url, nil)
	if err != nil {
		log.Printf("Initialize http client failed : %s\n", err)
		panic(err)
	}

	realIP := GetHostIp()
	request.Header.Set("X-Real-IP", realIP)

	return request
}

func GetHostIp() string {
	addrList, err := net.InterfaceAddrs()
	if err != nil {
		log.Printf("get host real ip  failed : %s\n", err)
		panic(err)
	}

	var ip string
	for _, address := range addrList {
		if ipNet, ok := address.(*net.IPNet); ok && !ipNet.IP.IsLoopback() {
			if ipNet.IP.To4() != nil {
				ip = ipNet.IP.String()
				break
			}
		}
	}

	return ip
}

func SliceString2BashString(src []string) (dst string) {
	dst = ""
	for num, command := range src {
		if num == len(src)-1 {
			dst = dst + command
		} else {
			dst = dst + command + " && "
		}
	}
	fmt.Println(dst)
	return dst
}
