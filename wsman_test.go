package gowsman

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"testing"

	"github.com/google/uuid"
	"github.com/icholy/digest"
)

func TestParseWsman(t *testing.T) {
	wsman := new(WSMan)
	f, err := os.Open("test/data/gensettings.xml")
	if err != nil {
		t.Fail()
	}
	ws, err := wsman.ParseWsman(bufio.NewReader(f))
	js, _ := json.MarshalIndent(ws, "", "   ")
	t.Logf("%s\n", string(js))
}

func TestGetAMTGeneralSettings(t *testing.T) {
	cfg_bytes, err := ioutil.ReadFile("test/data/cred.json")
	if err != nil {
		t.Logf(err.Error())
		t.Fail()
	}
	var cfg = map[string]string{}
	json.Unmarshal(cfg_bytes, &cfg)

	client := http.Client{
		Transport: &digest.Transport{
			Username: cfg["Username"],
			Password: cfg["Password"],
		},
	}
	url := fmt.Sprintf("http://%s:%s/wsman", cfg["Host"], cfg["Port"])
	wsman := new(WSMan)
	n_uuid := uuid.New()
	wsman_get_msg := wsman.CreateWsmanGet("AMT_GeneralSettings", n_uuid.String())
	resp, err := client.Post(url, "text/xml", bytes.NewBuffer([]byte(wsman_get_msg)))
	if err != nil {
		t.Logf("Error: %s\n", err)
		return
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	ws, err := wsman.ParseWsman(bytes.NewBuffer([]byte(body)))
	js, _ := json.MarshalIndent(ws.Body, "", "   ")
	t.Logf("%s\n", string(js))
}

func TestEnumPullCIMSoftwareIdentity(t *testing.T) {
	cfg_bytes, err := ioutil.ReadFile("test/data/cred.json")
	if err != nil {
		t.Logf(err.Error())
		t.Fail()
	}
	var cfg = map[string]string{}
	json.Unmarshal(cfg_bytes, &cfg)

	client := http.Client{
		Transport: &digest.Transport{
			Username: cfg["Username"],
			Password: cfg["Password"],
		},
	}
	url := fmt.Sprintf("http://%s:%s/wsman", cfg["Host"], cfg["Port"])
	wsman := new(WSMan)
	n_uuid := uuid.New()
	wsman_get_msg := wsman.CreateWsmanEnumerate("CIM_SoftwareIdentity", n_uuid.String())
	resp, err := client.Post(url, "text/xml", bytes.NewBuffer([]byte(wsman_get_msg)))
	if err != nil {
		t.Logf("Error: %s\n", err)
		return
	}
	body, err := ioutil.ReadAll(resp.Body)
	resp.Body.Close()
	ws, err := wsman.ParseWsman(bytes.NewBuffer([]byte(body)))
	enum_resp := ws.Body["EnumerateResponse"]
	enum_ctx := enum_resp.(map[string]interface{})["EnumerationContext"]
	if enum_ctx == nil {
		t.Log("No enumeration context returned")
		t.Fail()
		return
	}
	var enum_result []interface{}
	for {
		n_uuid = uuid.New()
		wsman_pull_msg := wsman.CreateWsmanPull("CIM_SoftwareIdentity", n_uuid.String(), fmt.Sprint(enum_ctx))
		resp, err = client.Post(url, "text/xml", bytes.NewBuffer([]byte(wsman_pull_msg)))
		if err != nil {
			t.Logf("Error: %s\n", err)
			return
		}
		body, err = ioutil.ReadAll(resp.Body)
		resp.Body.Close()
		ws, err = wsman.ParseWsman(bytes.NewBuffer([]byte(body)))
		pull_resp := ws.Body["PullResponse"]
		if pull_resp == nil {
			break
		}
		items := pull_resp.(map[string]interface{})["Items"]
		enum_result = append(enum_result, items.(map[string]interface{})["CIM_SoftwareIdentity"])
	}
	js, _ := json.MarshalIndent(enum_result, "", "   ")
	t.Logf("%s\n", string(js))

}
