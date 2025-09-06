// Package esx provides Elasticsearch client and indexing functionality
package esx

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"fiber-ent-apollo-pg/internal/config"

	es8 "github.com/elastic/go-elasticsearch/v8"
	"github.com/elastic/go-elasticsearch/v8/esapi"
	"github.com/samber/lo"
)

// Client is an alias for Elasticsearch v8 client
type Client = es8.Client

// Open creates a new Elasticsearch client based on configuration
func Open(cfg *config.Config) (*Client, func(), error) {
	if strings.TrimSpace(cfg.ES.Addrs) == "" {
		return nil, func() {}, nil
	}
	raw := strings.Split(cfg.ES.Addrs, ",")
	addrs := lo.FilterMap(raw, func(s string, _ int) (string, bool) {
		t := strings.TrimSpace(s)
		return t, t != ""
	})
	es, err := es8.NewClient(es8.Config{Addresses: addrs, Username: cfg.ES.Username, Password: cfg.ES.Password})
	if err != nil {
		return nil, func() {}, err
	}
	return es, func() {}, nil
}

// PostDoc represents a post document for Elasticsearch indexing
type PostDoc struct {
	ID        int    `json:"id"`
	Title     string `json:"title"`
	Content   string `json:"content"`
	UserID    int    `json:"user_id"`
	CreatedAt string `json:"created_at"`
}

// IndexPost indexes a post document into Elasticsearch
func IndexPost(_ context.Context, es *Client, index string, doc PostDoc) error {
	if es == nil {
		return nil
	}
	b, _ := json.Marshal(doc)
	res, err := es.Index(index, bytes.NewReader(b), es.Index.WithDocumentID(strconvItoa(doc.ID)))
	if err != nil {
		return err
	}
	defer func() {
		_ = res.Body.Close()
	}()
	if res.StatusCode >= http.StatusBadRequest {
		return fmtError(res)
	}
	return nil
}

// SearchPosts searches for posts in Elasticsearch index
func SearchPosts(_ context.Context, es *Client, index string, query string, from, size int) (map[string]any, error) {
	if es == nil {
		return map[string]any{"hits": []any{}}, nil
	}
	q := map[string]any{"query": map[string]any{"multi_match": map[string]any{"query": query, "fields": []string{"title^2", "content"}}}}
	b, _ := json.Marshal(q)
	res, err := es.Search(es.Search.WithIndex(index), es.Search.WithBody(bytes.NewReader(b)), es.Search.WithFrom(from), es.Search.WithSize(size))
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = res.Body.Close()
	}()
	if res.StatusCode >= http.StatusBadRequest {
		return nil, fmtError(res)
	}
	var out map[string]any
	_ = json.NewDecoder(res.Body).Decode(&out)
	return out, nil
}

// helpers
func strconvItoa(i int) string           { return strconv.Itoa(i) }
func fmtError(res *esapi.Response) error { return fmt.Errorf("es error: %s", res.String()) }
