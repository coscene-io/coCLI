package registry

import "testing"

func TestInferRegistryHost(t *testing.T) {
	tests := []struct {
		name     string
		endpoint string
		override string
		want     string
		wantErr  bool
	}{
		{
			name:     "prod endpoint",
			endpoint: "https://openapi.coscene.cn",
			want:     "cr.coscene.cn",
		},
		{
			name:     "staging endpoint",
			endpoint: "https://openapi.staging.coscene.cn",
			want:     "cr.staging.coscene.cn",
		},
		{
			name:     "dev endpoint",
			endpoint: "https://openapi.api.coscene.dev",
			want:     "cr.dev.coscene.cn",
		},
		{
			name:     "dev alias",
			endpoint: "https://api.dev.coscene.cn",
			want:     "cr.dev.coscene.cn",
		},
		{
			name:     "openapi prefix fallback",
			endpoint: "https://openapi.foo.bar",
			want:     "cr.foo.bar",
		},
		{
			name:     "explicit override",
			endpoint: "https://whatever",
			override: "custom.registry",
			want:     "custom.registry",
		},
		{
			name:     "unrecognized host",
			endpoint: "https://example.com",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := inferRegistryHost(tt.endpoint, tt.override)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Fatalf("want %q, got %q", tt.want, got)
			}
		})
	}
}
