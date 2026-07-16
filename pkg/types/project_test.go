package types

import "testing"

func TestGetImageNameWithTag(t *testing.T) {
	tests := []struct {
		name  string
		image ImageSpec
		env   EnvSpec
		tag   string
		want  string
	}{
		{
			name:  "prefix from image",
			image: ImageSpec{Name: "api", Repo: "api", Prefix: "reg.io/team"},
			tag:   "v1.2.3",
			want:  "reg.io/team/api:v1.2.3",
		},
		{
			name:  "prefix from env",
			image: ImageSpec{Name: "api", Repo: "api"},
			env:   EnvSpec{ImagePrefix: "reg.io/team"},
			tag:   "v1.2.3",
			want:  "reg.io/team/api:v1.2.3",
		},
		{
			name:  "no prefix",
			image: ImageSpec{Name: "api", Repo: "api"},
			tag:   "v1.2.3",
			want:  "api:v1.2.3",
		},
		{
			name:  "repo falls back to name",
			image: ImageSpec{Name: "worker", Prefix: "reg.io"},
			tag:   "latest",
			want:  "reg.io/worker:latest",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.image.GetImageNameWithTag(tt.env, tt.tag); got != tt.want {
				t.Fatalf("GetImageNameWithTag() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestGetImageNameTagPrecedence(t *testing.T) {
	tests := []struct {
		name  string
		image ImageSpec
		env   EnvSpec
		want  string
	}{
		{
			name:  "image tag wins over env tag",
			image: ImageSpec{Name: "api", Repo: "api", Tag: "v1"},
			env:   EnvSpec{ImageTag: "v2"},
			want:  "api:v1",
		},
		{
			name:  "env tag used when image tag empty",
			image: ImageSpec{Name: "api", Repo: "api"},
			env:   EnvSpec{ImageTag: "v2"},
			want:  "api:v2",
		},
		{
			name:  "defaults to latest",
			image: ImageSpec{Name: "api", Repo: "api"},
			want:  "api:latest",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.image.GetImageName(tt.env); got != tt.want {
				t.Fatalf("GetImageName() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestGetImageNameLatestGuard(t *testing.T) {
	// When the primary tag resolves to "latest", the :latest sibling equals it.
	// This is exactly the condition the build-image no-op guard relies on.
	image := ImageSpec{Name: "api", Repo: "api", Prefix: "reg.io"}
	env := EnvSpec{} // no tag anywhere -> defaults to "latest"
	if primary, latest := image.GetImageName(env), image.GetImageNameWithTag(env, "latest"); primary != latest {
		t.Fatalf("expected primary %q to equal latest sibling %q", primary, latest)
	}
}
