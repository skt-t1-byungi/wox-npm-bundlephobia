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
    Result []jsonRPCResultItem
}

type jsonRPCResultItem struct {
    Title    string          `json:"Title"`
    SubTitle string          `json:"SubTitle"`
    IcoPath  string          `json:"IcoPath"`
    Action   *jsonRPCRequest `json:"JsonRPCAction,omitempty"`
}

type suggestion struct {
    Name        string `json:"name"`
    Version     string `json:"version"`
    Description string `json:"description"`
    Publisher   struct {
        Username string `json:"username"`
        Email    string `json:"email"`
    } `json:"publisher"`
}

type detail struct {
    Name        string `json:"name"`
    Description string `json:"description"`
    Version     string `json:"version"`
    DepCount    uint   `json:"dependencyCount"`
    Gzip        uint64 `json:"gzip"`
    Size        uint64 `json:"size"`
    Error       *struct {
        Code    string   `json:"code"`
        Message string   `json:"message"`
        Details struct{} `json:"details"`
    } `json:"error,omitempty"`
}

func main() {
    req, err := parseJsonRPCRequest(os.Args[1])
    if err != nil {
        return
    }

    if req.Method == "openBrowser" {
        _ = open.Start("https://bundlephobia.com/result?p=" + req.Parameters[0].(string))
        return
    }

    q := strings.TrimSpace(req.Parameters[0].(string))
    if q[len(q)-1] == '!' {
        showDetail(strings.TrimRight(q, "!"))
    } else {
        suggestPkg(q)
    }
}

func parseJsonRPCRequest(str string) (*jsonRPCRequest, error) {
    var ret *jsonRPCRequest
    if err := json.Unmarshal([]byte(str), &ret); err != nil {
        return nil, err
    }
    return ret, nil
}

func sendQueryResult(result []jsonRPCResultItem) error {
    b, err := json.Marshal(jsonRPCResponse{Result: result})
    if err != nil {
        return err
    }
    _, err = os.Stdout.Write(b)
    return err
}

func suggestPkg(q string) {
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

    items := make([]jsonRPCResultItem, len(suggestions))
    for idx, suggestion := range suggestions {
        items[idx] = jsonRPCResultItem{
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
    _ = sendQueryResult(items)
}

func showDetail(q string) {
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
        _ = sendQueryResult([]jsonRPCResultItem{
            {
                Title:    "Not found : " + q,
                SubTitle: "The package you were looking for doesn't exist.",
                IcoPath:  "icon.png",
            },
        })
        return
    }
    _ = sendQueryResult([]jsonRPCResultItem{
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
