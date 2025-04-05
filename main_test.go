package main_test

import (
	"net/url"
	"reflect"
	"testing"

	main "github.com/dev-shimada/api-stubs"
)

func Test_pathMatcher(t *testing.T) {
	type args struct {
		endpoint   main.Endpoint
		gotRawPath string
		gotPath    string
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "url",
			args: args{
				endpoint: main.Endpoint{
					Request: main.Request{
						URL: "http://example.com/path",
					},
				},
				gotRawPath: "http://example.com/path/",
			},
			want: true,
		},
		{
			name: "url false",
			args: args{
				endpoint: main.Endpoint{
					Request: main.Request{
						URL: "http://example.com/path",
					},
				},
				gotRawPath: "http://example.com/path?a=1",
			},
			want: false,
		},
		{
			name: "urlPattern",
			args: args{
				endpoint: main.Endpoint{
					Request: main.Request{
						URLPattern: "http://example.com/(\\d{5})/",
					},
				},
				gotRawPath: "http://example.com/123456",
			},
			want: true,
		},
		{
			name: "urlPattern false",
			args: args{
				endpoint: main.Endpoint{
					Request: main.Request{
						URLPattern: "http://example.com/(\\d{5})/",
					},
				},
				gotRawPath: "http://example.com/abcde",
			},
			want: false,
		},
		{
			name: "urlPath",
			args: args{
				endpoint: main.Endpoint{
					Request: main.Request{
						URLPath: "http://example.com/path",
					},
				},
				gotPath: "http://example.com/path",
			},
			want: true,
		},
		{
			name: "urlPath false",
			args: args{
				endpoint: main.Endpoint{
					Request: main.Request{
						URLPath: "http://example.com/path",
					},
				},
				gotPath: "http://example.com/path/abc",
			},
			want: false,
		},
		{
			name: "urlPathPattern",
			args: args{
				endpoint: main.Endpoint{
					Request: main.Request{
						URLPathPattern: "http://example.com/(\\d{5})",
					},
				},
				gotPath: "http://example.com/12345/abc",
			},
			want: true,
		},
		{
			name: "urlPathPattern false",
			args: args{
				endpoint: main.Endpoint{
					Request: main.Request{
						URLPathPattern: "http://example.com/(\\d{5})",
					},
				},
				gotPath: "http://example.com/abc",
			},
			want: false,
		},
		{
			name: "urlPathTemplate",
			args: args{
				endpoint: main.Endpoint{
					Request: main.Request{
						URLPathTemplate: "http://example.com/{path1}/{path2}/{path3}/{path4}/{path5}",
						PathParameters: map[string]main.Matcher{
							"path1": {
								EqualTo: "12345",
							},
							"path2": {
								Matches: "[0-9]{5}",
							},
							"path3": {
								DoesNotMatch: "[a-z]{5}",
							},
							"path4": {
								Contains: "abc",
							},
							"path5": {
								DoesNotContain: "xyz",
							},
						},
					},
				},
				gotPath: "http://example.com/12345/67890/00000/abcxyz/12345",
			},
			want: true,
		},
		{
			name: "urlPathTemplate equalTo false",
			args: args{
				endpoint: main.Endpoint{
					Request: main.Request{
						URLPathTemplate: "http://example.com/{path}",
						PathParameters: map[string]main.Matcher{
							"path": {
								EqualTo: "12345",
							},
						},
					},
				},
				gotPath: "http://example.com/123456",
			},
			want: false,
		},
		{
			name: "urlPathTemplate contains false",
			args: args{
				endpoint: main.Endpoint{
					Request: main.Request{
						URLPathTemplate: "http://example.com/{path}",
						PathParameters: map[string]main.Matcher{
							"path": {
								Contains: "12345",
							},
						},
					},
				},
				gotPath: "http://example.com/1234",
			},
			want: false,
		},
		{
			name: "urlPathTemplate doesNotContain false",
			args: args{
				endpoint: main.Endpoint{
					Request: main.Request{
						URLPathTemplate: "http://example.com/{path}",
						PathParameters: map[string]main.Matcher{
							"path": {
								DoesNotContain: "12345",
							},
						},
					},
				},
				gotPath: "http://example.com/12345abc",
			},
			want: false,
		},
		{
			name: "urlPathTemplate matches false",
			args: args{
				endpoint: main.Endpoint{
					Request: main.Request{
						URLPathTemplate: "http://example.com/{path}",
						PathParameters: map[string]main.Matcher{
							"path": {
								Matches: "[0-9]{5}",
							},
						},
					},
				},
				gotPath: "http://example.com/1234",
			},
			want: false,
		},
		{
			name: "urlPathTemplate doesNotMatch false",
			args: args{
				endpoint: main.Endpoint{
					Request: main.Request{
						URLPathTemplate: "http://example.com/{path}",
						PathParameters: map[string]main.Matcher{
							"path": {
								DoesNotMatch: "[0-9]{5}",
							},
						},
					},
				},
				gotPath: "http://example.com/12345abc",
			},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := main.ExportPathMatcher(tt.args.endpoint, tt.args.gotRawPath, tt.args.gotPath); got != tt.want {
				t.Errorf("pathMatcher() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_queryMatcher(t *testing.T) {
	type args struct {
		endpoint main.Endpoint
		gotQuery url.Values
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := main.ExportQueryMatcher(tt.args.endpoint, tt.args.gotQuery); got != tt.want {
				t.Errorf("queryMatcher() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_loadConfig(t *testing.T) {
	type args struct {
		filePath string
	}
	tests := []struct {
		name    string
		args    args
		want    []main.Endpoint
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := main.ExportLoadConfig(tt.args.filePath)
			if (err != nil) != tt.wantErr {
				t.Errorf("loadConfig() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("loadConfig() = %v, want %v", got, tt.want)
			}
		})
	}
}
