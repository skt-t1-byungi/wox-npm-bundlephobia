package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"

	human "github.com/dustin/go-humanize"
	"github.com/skratchdot/open-golang/open"
)

type jsonRPCRequest struct {
	Persist    bool          `json:"DontHideAfterAction"`
	Method     string        `json:"method"`
	Parameters []interface{} `json:"parameters"`
}

type jsonRPCResponse struct {
	Result []resultItem
}

type resultItem struct {
	Title    string          `json:"Title"`
	SubTitle string          `json:"SubTitle"`
	IcoPath  string          `json:"IcoPath"`
	Action   *jsonRPCRequest `json:"JsonRPCAction,omitempty"`
}

type suggestion struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

type detail struct {
	Gzip  uint64      `json:"gzip"`
	Size  uint64      `json:"size"`
	Error interface{} `json:"error,omitempty"`
}

func main() {
	req, err := parseRPCRequest(os.Args[1])
	if err != nil {
		return
	}

	if req.Method == "openBrowser" {
		_ = open.Start("https://bundlephobia.com/result?p=" + req.Parameters[0].(string))
		return
	}

	q := strings.TrimSpace(req.Parameters[0].(string))
	if q[len(q)-1] == '!' {
		handleDetail(strings.TrimRight(q, "!"))
	} else {
		handlePkgSuggestion(q)
	}
}

func parseRPCRequest(str string) (*jsonRPCRequest, error) {
	var ret *jsonRPCRequest
	if err := json.Unmarshal([]byte(str), &ret); err != nil {
		return nil, err
	}
	return ret, nil
}

func sendRPCResultItems(result []resultItem) error {
	b, err := json.Marshal(jsonRPCResponse{Result: result})
	if err != nil {
		return err
	}
	_, err = os.Stdout.Write(b)
	return err
}

func handlePkgSuggestion(q string) {
	if len(q) < 2 {
		return
	}

	resp, err := http.Get("https://www.npmjs.com/search/suggestions?q=" + q)
	if err != nil {
		return
	}
	defer resp.Body.Close()

	var suggestions []suggestion
	if err = json.NewDecoder(resp.Body).Decode(&suggestions); err != nil {
		return
	}

	items := make([]resultItem, len(suggestions))
	for idx, suggestion := range suggestions {
		items[idx] = resultItem{
			Title:    suggestion.Name,
			SubTitle: suggestion.Description,
			IcoPath:  "icon.png",
			Action: &jsonRPCRequest{
				Persist:    true,
				Method:     "Wox.ChangeQuery",
				Parameters: []interface{}{"nbp " + suggestion.Name + "!", true},
			},
		}
	}
	_ = sendRPCResultItems(items)
}

func handleDetail(q string) {
	resp, err := http.Get("https://bundlephobia.com/api/size?package=" + q)
	if err != nil {
		return
	}
	defer resp.Body.Close()

	var detail detail
	if err = json.NewDecoder(resp.Body).Decode(&detail); err != nil {
		return
	}

	if detail.Error != nil {
		_ = sendRPCResultItems([]resultItem{
			{
				Title:    "Not found : " + q,
				SubTitle: "The package you were looking for doesn't exist.",
				IcoPath:  "icon.png",
			},
		})
		return
	}

	_ = sendRPCResultItems([]resultItem{
		{
			Title:    fmt.Sprintf("minified: %s, gzipped: %s", human.Bytes(detail.Size), human.Bytes(detail.Gzip)),
			SubTitle: "Open your browser for more information.",
			IcoPath:  "icon.png",
			Action: &jsonRPCRequest{
				Method:     "openBrowser",
				Parameters: []interface{}{q},
			},
		},
	})
}
