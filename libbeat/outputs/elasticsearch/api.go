// Licensed to Elasticsearch B.V. under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. Elasticsearch B.V. licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package elasticsearch

import (
	"encoding/json"

	"github.com/pkg/errors"
	"github.com/elastic/beats/libbeat/logp"
)

// QueryResult contains the result of a query.
type QueryResult struct {
	Ok           bool            `json:"ok"`
	Index        string          `json:"_index"`
	Type         string          `json:"_type"`
	ID           string          `json:"_id"`
	Source       json.RawMessage `json:"_source"`
	Version      int             `json:"_version"`
	Exists       bool            `json:"exists"`
	Found        bool            `json:"found"`   // Only used prior to ES 6. You must also check for Result == "found".
	Created      bool            `json:"created"` // Only used prior to ES 6. You must also check for Result == "created".
	Result       string          `json:"result"`  // Only used in ES 6+.
	Acknowledged bool            `json:"acknowledged"`
	Matches      []string        `json:"matches"`
}

// SearchResults contains the results of a search.
type SearchResults struct {
	Took   int                        `json:"took"`
	Shards json.RawMessage            `json:"_shards"`
	Hits   Hits                       `json:"hits"`
	Aggs   map[string]json.RawMessage `json:"aggregations"`
}

// Hits contains the hits.
type Hits struct {
	Total int
	Hits  []json.RawMessage `json:"hits"`
}

// CountResults contains the count of results.
type CountResults struct {
	Count  int             `json:"count"`
	Shards json.RawMessage `json:"_shards"`
}

func withQueryResult(status int, resp []byte, err error) (int, *QueryResult, error) {
	if err != nil {
		return status, nil, errors.Wrapf(err, "Elasticsearch response: %s", resp)
	}
	result, err := readQueryResult(resp)
	return status, result, err
}

func readQueryResult(obj []byte) (*QueryResult, error) {
	var result QueryResult
	if obj == nil {
		return nil, nil
	}

	err := json.Unmarshal(obj, &result)
	if err != nil {
		return nil, err
	}
	return &result, err
}

func readSearchResult(obj []byte) (*SearchResults, error) {
	var result SearchResults
	if obj == nil {
		return nil, nil
	}

	err := json.Unmarshal(obj, &result)
	if err != nil {
		return nil, err
	}
	return &result, err
}

func readCountResult(obj []byte) (*CountResults, error) {
	if obj == nil {
		return nil, nil
	}

	var result CountResults
	err := json.Unmarshal(obj, &result)
	if err != nil {
		return nil, err
	}
	return &result, err
}

// Index adds or updates a typed JSON document in a specified index, making it
// searchable. In case id is empty, a new id is created over a HTTP POST request.
// Otherwise, a HTTP PUT request is issued.
// Implements: http://www.elastic.co/guide/en/elasticsearch/reference/current/docs-index_.html
func (es *Connection) Index(
	index, docType, id string,
	params map[string]string,
	body interface{},
) (int, *QueryResult, error) {
	method := "PUT"
	if id == "" {
		method = "POST"
	}
	return withQueryResult(es.apiCall(method, index, docType, id, "", params, body))
}

// Ingest pushes a pipeline of updates.
func (es *Connection) Ingest(
	index, docType, pipeline, id string,
	params map[string]string,
	body interface{},
) (int, *QueryResult, error) {
	method := "PUT"
	if id == "" {
		method = "POST"
	}
	return withQueryResult(es.apiCall(method, index, docType, id, pipeline, params, body))
}

// Refresh an index. Call this after doing inserts or creating/deleting
// indexes in unit tests.
func (es *Connection) Refresh(index string) (int, *QueryResult, error) {
	return withQueryResult(es.apiCall("POST", index, "", "_refresh", "", nil, nil))
}

// CreateIndex creates a new index, optionally with settings and mappings passed in
// the body.
// Implements: https://www.elastic.co/guide/en/elasticsearch/reference/current/indices-create-index.html
//
func (es *Connection) CreateIndex(index string, body interface{}) (int, *QueryResult, error) {
	logp.Info("[api 150] current index is: %s, and body is %s", index, body)
	return withQueryResult(es.apiCall("PUT", index, "", "", "", nil, body))
}

// IndexExists checks if an index exists.
// Implements: https://www.elastic.co/guide/en/elasticsearch/reference/current/indices-exists.html
//
func (es *Connection) IndexExists(index string) (int, error) {
	status, _, err := es.apiCall("HEAD", index, "", "", "", nil, nil)
	return status, err
}

// Delete deletes a typed JSON document from a specific index based on its id.
// Implements: http://www.elastic.co/guide/en/elasticsearch/reference/current/docs-delete.html
func (es *Connection) Delete(index string, docType string, id string, params map[string]string) (int, *QueryResult, error) {
	return withQueryResult(es.apiCall("DELETE", index, docType, id, "", params, nil))
}

// CreatePipeline create a new ingest pipeline with name id.
// Implements: https://www.elastic.co/guide/en/elasticsearch/reference/current/put-pipeline-api.html
func (es *Connection) CreatePipeline(
	id string,
	params map[string]string,
	body interface{},
) (int, *QueryResult, error) {
	return withQueryResult(es.apiCall("PUT", "_ingest", "pipeline", id, "", params, body))
}

// DeletePipeline deletes an ingest pipeline by id.
// Implements: https://www.elastic.co/guide/en/elasticsearch/reference/current/delete-pipeline-api.html
func (es *Connection) DeletePipeline(
	id string,
	params map[string]string,
) (int, *QueryResult, error) {
	return withQueryResult(es.apiCall("DELETE", "_ingest", "pipeline", id, "", params, nil))
}

// SearchURI executes a search request using a URI by providing request parameters.
// Implements: http://www.elastic.co/guide/en/elasticsearch/reference/current/search-uri-request.html
func (es *Connection) SearchURI(index string, docType string, params map[string]string) (int, *SearchResults, error) {
	return es.SearchURIWithBody(index, docType, params, nil)
}

// SearchURIWithBody executes a search request using a URI by providing request
// parameters and a request body.
func (es *Connection) SearchURIWithBody(
	index string,
	docType string,
	params map[string]string,
	body interface{},
) (int, *SearchResults, error) {
	status, resp, err := es.apiCall("GET", index, docType, "_search", "", params, body)
	if err != nil {
		return status, nil, err
	}
	result, err := readSearchResult(resp)
	return status, result, err
}

// CountSearchURI counts the results for a search request.
func (es *Connection) CountSearchURI(
	index string, docType string,
	params map[string]string,
) (int, *CountResults, error) {
	status, resp, err := es.apiCall("GET", index, docType, "_count", "", params, nil)
	if err != nil {
		return status, nil, err
	}
	result, err := readCountResult(resp)
	return status, result, err
}

func (es *Connection) apiCall(
	method, index, docType, id, pipeline string,
	params map[string]string,
	body interface{},
) (int, []byte, error) {
	path, err := makePath(index, docType, id)
	if err != nil {
		return 0, nil, err
	}
	return es.Request(method, path, pipeline, params, body)
}
