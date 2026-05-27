package main

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
)

func toggleSSID(client *http.Client, routerURL string, enable bool, config Config) error {
	getPath := fmt.Sprintf("/template.gch?pid=1002&nextpage=net_wlanm_essid1_t.gch&IF_VIEWID=IGD.LD1.WLAN%s", config.WlanIndex)
	fullGetURL := strings.TrimRight(routerURL, "/") + getPath

	getReq, err := http.NewRequest("GET", fullGetURL, nil)
	if err != nil {
		return fmt.Errorf("failed to create GET request: %v", err)
	}
	getReq.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64)")

	getResp, err := client.Do(getReq)
	if err != nil {
		return fmt.Errorf("failed to fetch WLAN page: %v", err)
	}
	defer getResp.Body.Close()

	bodyBytes, err := io.ReadAll(getResp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response: %v", err)
	}
	htmlContent := string(bodyBytes)

	formData := url.Values{}
	configKeys := make(map[string]bool)

	sessionTokenRe := regexp.MustCompile(`var\s+session_token\s*=\s*["']([^"']+)["']`)
	inputTagRe := regexp.MustCompile(`(?i)<input\s+([^>]+)>`)
	idRe := regexp.MustCompile(`(?i)(?:id|name)=["']([^"']+)["']`)
	valRe := regexp.MustCompile(`(?i)value=["']([^"']*)["']`)

	nullRe := regexp.MustCompile(`\bvar\s+CONFIG_PARA\s*=\s*new\s+Array\(\s*(["'][^"']*["'](?:\s*,\s*["'][^"']*["'])*)\s*\)`)
	for _, match := range nullRe.FindAllStringSubmatch(htmlContent, -1) {
		items := strings.Split(match[1], ",")
		for _, item := range items {
			key := strings.Trim(strings.TrimSpace(item), "\"'")
			if key != "" {
				configKeys[key] = true
				formData.Set(key, "NULL")
			}
		}
	}

	for _, tagMatch := range inputTagRe.FindAllStringSubmatch(htmlContent, -1) {
		attrStr := tagMatch[1]
		idMatch := idRe.FindStringSubmatch(attrStr)
		valMatch := valRe.FindStringSubmatch(attrStr)

		if len(idMatch) > 1 {
			key := idMatch[1]
			if strings.HasPrefix(key, "Frm_") || key == "logout" || key == "temClickURL" || key == "Submit" {
				continue
			}

			if configKeys[key] {
				continue
			}

			val := ""
			if len(valMatch) > 1 {
				val = valMatch[1]
			}
			formData.Set(key, val)
		}
	}

	jsRe := regexp.MustCompile(`(?i)\b(?:Transfer_meaning|setValue)\s*\(\s*['"]([^'"]+)['"]\s*,\s*['"]((?:\\[\s\S]|[^'"])*)['"]\s*\)`)
	for _, jsMatch := range jsRe.FindAllStringSubmatch(htmlContent, -1) {
		if len(jsMatch) == 3 {
			key := jsMatch[1]
			rawVal := jsMatch[2]

			if strings.HasPrefix(key, "Frm_") || configKeys[key] {
				continue
			}

			val := unescapeJavaScriptString(rawVal)

			formData.Set(key, val)
		}
	}

	if matches := sessionTokenRe.FindStringSubmatch(htmlContent); len(matches) > 1 {
		formData.Set("_SESSION_TOKEN", matches[1])
	}

	formData.Del("IF_ACTIONv6")
	formData.Del("IF_UPLOADING")
	formData.Set("IF_ACTION", "apply")
	formData.Set("IF_ERRORSTR", "SUCC")
	formData.Set("IF_ERRORPARAM", "SUCC")
	formData.Set("IF_ERRORTYPE", "-1")
	formData.Set("IF_CONFIGTAG", "Y")

	if enable {
		formData.Set("Enable", "1")
		formData.Set("ESSID", config.WlanESSID)
		formData.Set("ESSIDHideEnable", config.WlanESSIDHideEnable)
		formData.Set("MaxUserNum", config.WlanMaxUserNum)
		formData.Set("Priority", config.WlanPriority)
		formData.Set("VapIsolationEnable", config.WlanVapIsolationEnable)
	} else {
		formData.Set("Enable", "0")
		formData.Set("ESSID", "NULL")
		formData.Set("ESSIDHideEnable", "NULL")
		formData.Set("MaxUserNum", "NULL")
		formData.Set("Priority", "0")
		formData.Set("VapIsolationEnable", "NULL")
	}


	postPath := "/getpage.gch?pid=1002&nextpage=net_wlanm_essid1_t.gch"
	fullPostUrl := strings.TrimRight(routerURL, "/") + postPath

	postReq, err := http.NewRequest("POST", fullPostUrl, strings.NewReader(formData.Encode()))
	if err != nil {
		return fmt.Errorf("failed to create POST request: %v", err)
	}

	postResp, err := client.Do(postReq)
	if err != nil {
		return fmt.Errorf("failed to send apply request: %v", err)
	}
	defer postResp.Body.Close()

	if postResp.StatusCode == 200 || postResp.StatusCode == 302 {
		state := "disabled"
		if enable {
			state = "enabled"
		}
		fmt.Printf("Successfully requested SSID to be %s\n", state)
		return nil
	}

	return fmt.Errorf("failed to apply settings, unexpected status code: %d", postResp.StatusCode)
}

func unescapeJavaScriptString(s string) string {
	if !strings.Contains(s, `\`) {
		return s
	}
	quoted := `"` + s + `"`
	unquoted, err := strconv.Unquote(quoted)
	if err != nil {
		return s
	}
	return unquoted
}
