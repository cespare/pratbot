package authutil

import (
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"sort"
	"strconv"
	"strings"
	"time"
)

type ByPair [][]string

func (p ByPair) Len() int           { return len(p) }
func (p ByPair) Less(i, j int) bool { return p[i][0] < p[j][0] }
func (p ByPair) Swap(i, j int)      { p[i], p[j] = p[j], p[i] }

func prepareQueryString(params map[string]string, exclude []string) string {
	var result [][]string
	for k, v := range params {
		excluded := false
		for _, e := range exclude {
			if k == e {
				excluded = true
				break
			}
		}
		if !excluded {
			result = append(result, []string{k, v})
		}
	}
	sort.Sort(ByPair(result))
	var pairs []string
	for _, p := range result {
		pairs = append(pairs, strings.Join(p, "="))
	}
	return strings.Join(pairs, "")
}

func Signature(secret, method, path, body string, params map[string]string) string {
	exclude := []string{"signature"}
	signature := secret + strings.ToUpper(method) + path + prepareQueryString(params, exclude) + body
	h := sha256.New()
	h.Write([]byte(signature))
	var buf bytes.Buffer
	enc := base64.NewEncoder(base64.URLEncoding, &buf)
	enc.Write(h.Sum(nil))
	enc.Close()
	return buf.String()[:43]
}

// Route needs leading slash, no trailing ?. Example: '/eventhub'
// TODO: Make this less fragile.
func SignRequest(route, apiKey, secret string) string {
	expires := int(time.Now().Unix() + 300)
	params := map[string]string{"api_key": apiKey, "expires": strconv.Itoa(expires)}
	signature := Signature(secret, "GET", route, "", params)
	params["signature"] = signature
	var kv []string
	for k, v := range params {
		kv = append(kv, k+"="+v)
	}
	return route + "?" + strings.Join(kv, "&")
}
