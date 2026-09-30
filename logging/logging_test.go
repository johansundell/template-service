package logging

import "testing"

type recorder struct{ n int }

func (r *recorder) Infof(string, ...interface{})    { r.n++ }
func (r *recorder) Warningf(string, ...interface{}) { r.n++ }
func (r *recorder) Errorf(string, ...interface{})   { r.n++ }

func TestOrStd(t *testing.T) {
	if _, ok := OrStd(nil).(Std); !ok {
		t.Error("expected Std for a nil logger")
	}
	r := &recorder{}
	OrStd(r).Infof("x")
	if r.n != 1 {
		t.Error("expected the given logger to be used")
	}
}
