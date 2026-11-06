package observe

import (
	"testing"

	"github.com/Controlwatch/control-watch/pkg/model"
	"github.com/Controlwatch/control-watch/pkg/surface"
)

func TestClassifyFunctions(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		functions []surface.FunctionName
		want      model.Classification
	}{
		{
			name:      "RWA",
			functions: append(append([]surface.FunctionName{}, sep41Signature()...), rwaSignature()...),
			want:      model.ClassSEP57RWA,
		},
		{name: "SEP-41", functions: sep41Signature(), want: model.ClassSEP41},
		{name: "unknown", functions: []surface.FunctionName{surface.FunctionPaused}, want: model.ClassUnknown},
		{
			name:      "incomplete RWA remains SEP-41",
			functions: append(append([]surface.FunctionName{}, sep41Signature()...), rwaSignature()[:len(rwaSignature())-1]...),
			want:      model.ClassSEP41,
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			set := make(map[surface.FunctionName]struct{}, len(test.functions))
			for _, function := range test.functions {
				set[function] = struct{}{}
			}
			if got := classifyFunctions(set); got != test.want {
				t.Fatalf("classification = %q, want %q", got, test.want)
			}
		})
	}
}
