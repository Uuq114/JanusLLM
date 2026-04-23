package auth

import "testing"

func TestEffectiveModelList(t *testing.T) {
	tests := []struct {
		name string
		team StringSlice
		key  StringSlice
		want StringSlice
	}{
		{
			name: "team wildcard keeps key restriction",
			team: StringSlice{"*"},
			key:  StringSlice{"qwen", "minimax"},
			want: StringSlice{"qwen", "minimax"},
		},
		{
			name: "key wildcard inherits team restriction",
			team: StringSlice{"qwen", "minimax"},
			key:  StringSlice{"*"},
			want: StringSlice{"qwen", "minimax"},
		},
		{
			name: "intersection of team and key",
			team: StringSlice{"qwen", "minimax"},
			key:  StringSlice{"qwen", "claude"},
			want: StringSlice{"qwen"},
		},
		{
			name: "both wildcard remains wildcard",
			team: StringSlice{"*"},
			key:  StringSlice{"*"},
			want: StringSlice{"*"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := EffectiveModelList(tt.team, tt.key)
			if len(got) != len(tt.want) {
				t.Fatalf("expected %v, got %v", tt.want, got)
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Fatalf("expected %v, got %v", tt.want, got)
				}
			}
		})
	}
}
