package source

import (
	"encoding/base64"
	"reflect"
	"testing"
)

func TestGetSource(t *testing.T) {
	tests := []struct {
		url     string
		want    interface{}
		wantErr bool
	}{
		{"https://github.com/efinityhub/gitpatrol", &GitHubSource{}, false},
		{"https://gitlab.com/group/project", &GitLabSource{}, false},
		{"https://example.com/some/repo", nil, true},
	}

	for _, tt := range tests {
		t.Run(tt.url, func(t *testing.T) {
			got, err := GetSource(tt.url, "", "")
			if (err != nil) != tt.wantErr {
				t.Fatalf("err = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr && reflect.TypeOf(got) != reflect.TypeOf(tt.want) {
				t.Errorf("got %T, want %T", got, tt.want)
			}
		})
	}
}

func TestGitAuthHeaderArgs(t *testing.T) {
	if got := gitAuthHeaderArgs("github.com", "x-access-token", ""); got != nil {
		t.Errorf("no token must add no arguments, got %v", got)
	}

	got := gitAuthHeaderArgs("github.com", "x-access-token", "tok")
	creds := base64.StdEncoding.EncodeToString([]byte("x-access-token:tok"))
	want := []string{"-c", "http.https://github.com/.extraheader=AUTHORIZATION: basic " + creds}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestProviderAuthArgsAreScopedToTheirHost(t *testing.T) {
	gh, _ := GetSource("https://github.com/a/b", "ghtoken", "")
	gl, _ := GetSource("https://gitlab.com/a/b", "", "gltoken")

	if got := gh.GitAuthArgs(); len(got) != 2 || got[1] != "http.https://github.com/.extraheader=AUTHORIZATION: basic "+base64.StdEncoding.EncodeToString([]byte("x-access-token:ghtoken")) {
		t.Errorf("github args = %v", got)
	}
	if got := gl.GitAuthArgs(); len(got) != 2 || got[1] != "http.https://gitlab.com/.extraheader=AUTHORIZATION: basic "+base64.StdEncoding.EncodeToString([]byte("oauth2:gltoken")) {
		t.Errorf("gitlab args = %v", got)
	}

	noToken, _ := GetSource("https://github.com/a/b", "", "")
	if got := noToken.GitAuthArgs(); got != nil {
		t.Errorf("public repos must clone without auth args, got %v", got)
	}
}
