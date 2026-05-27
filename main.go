package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
)

type Config struct {
	RouterURL              string
	Username               string
	Password               string
	WlanIndex              string
	WlanESSID              string
	WlanESSIDHideEnable    string
	WlanMaxUserNum         string
	WlanPriority           string
	WlanVapIsolationEnable string
}

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Error: Missing required argument.")
		fmt.Printf("Usage: %s <true|false>\n", filepath.Base(os.Args[0]))
		return
	}

	enable, err := strconv.ParseBool(os.Args[1])
	if err != nil {
		fmt.Printf("Error: Invalid argument '%s'. Please provide 'true' or 'false'.\n", os.Args[1])
		return
	}

	if err := loadEnvFile(); err != nil {
		fmt.Printf("Note: %v\n", err)
	}

	config := Config{
		RouterURL:              getEnv("ROUTER_URL", nil),
		Username:               getEnv("ROUTER_USERNAME", nil),
		Password:               getEnv("ROUTER_PASSWORD", nil),
		WlanIndex:              getEnv("WLAN_INDEX", stringPtr("1")),
		WlanESSID:              getEnv("WLAN_ESSID", nil),
		WlanESSIDHideEnable:    getEnv("WLAN_ESSID_HIDE_ENABLE", stringPtr("0")),
		WlanMaxUserNum:         getEnv("WLAN_MAX_USER_NUM", stringPtr("32")),
		WlanPriority:           getEnv("WLAN_PRIORITY", stringPtr("0")),
		WlanVapIsolationEnable: getEnv("WLAN_VAP_ISOLATION_ENABLE", stringPtr("0")),
	}

	client, err := loginToRouter(config)
	if err != nil {
		fmt.Printf("Login failed: %v\n", err)
		return
	}

	err = toggleSSID(client, config.RouterURL, enable, config)
	if err != nil {
		fmt.Printf("Failed to toggle SSID: %v\n", err)
		return
	}
}
