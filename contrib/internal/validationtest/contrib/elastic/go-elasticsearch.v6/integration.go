// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016 Datadog, Inc.

package elastic6

import (
	"context"
	"strings"
	"testing"

	elastictrace "gopkg.in/DataDog/dd-trace-go.v1/contrib/elastic/go-elasticsearch.v6"

	elasticsearch6 "github.com/elastic/go-elasticsearch/v6"
	esapi6 "github.com/elastic/go-elasticsearch/v6/esapi"
	"github.com/stretchr/testify/require"
)

const (
	elasticV6URL = "http://127.0.0.1:9202"
	elasticV7URL = "http://127.0.0.1:9203"
	elasticV8URL = "http://127.0.0.1:9204"
)

type Integration struct {
	client   *elasticsearch6.Client
	numSpans int
	opts     []elastictrace.ClientOption
}

func New() *Integration {
	return &Integration{
		opts: make([]elastictrace.ClientOption, 0),
	}
}

func (i *Integration) Name() string {
	return "elastic/go-elasticsearch.v6"
}

func (i *Integration) Init(t *testing.T) {
	t.Helper()
	cfg := elasticsearch6.Config{
		Transport: elastictrace.NewRoundTripper(i.opts...),
		Addresses: []string{
			elasticV6URL,
		},
	}
	var err error
	i.client, err = elasticsearch6.NewClient(cfg)
	require.NoError(t, err)
	t.Cleanup(func() {
		i.numSpans = 0
	})
}

func (i *Integration) GenSpans(t *testing.T) {
	t.Helper()

	var err error
	_, err = esapi6.IndexRequest{
		Index:        "twitter",
		DocumentID:   "1",
		DocumentType: "tweet",
		Body:         strings.NewReader(`{"user": "test", "message": "hello"}`),
	}.Do(context.Background(), i.client)
	require.NoError(t, err)
	i.numSpans++

	_, err = esapi6.GetRequest{
		Index:        "twitter",
		DocumentID:   "1",
		DocumentType: "tweet",
	}.Do(context.Background(), i.client)
	require.NoError(t, err)
	i.numSpans++

	_, err = esapi6.GetRequest{
		Index:      "not-real-index",
		DocumentID: "1",
	}.Do(context.Background(), i.client)
	require.NoError(t, err)
	i.numSpans++
}

func (i *Integration) NumSpans() int {
	return i.numSpans
}

func (i *Integration) WithServiceName(name string) {
	i.opts = append(i.opts, elastictrace.WithServiceName(name))
}
