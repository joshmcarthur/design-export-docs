package rewrite

import "testing"

const id = "0f0998a4b02ec958ff53ecbd8f39bece"

func TestApply(t *testing.T) {
	r := New(map[string]string{id: "assets/Logos/a b~.svg"}, "/ds/vt/")
	in := `a{background:url(/_blob/` + id + `)} <img src="/_blob/` + id + `"> "/_blob/` +
		"ffffffffffffffffffffffffffffffff" + `"`
	got := r.Apply(in)
	want := `a{background:url(/ds/vt/assets/Logos/a%20b~.svg)} <img src="/ds/vt/assets/Logos/a%20b~.svg"> "/_blob/ffffffffffffffffffffffffffffffff"`
	if got != want {
		t.Errorf("got  %s\nwant %s", got, want)
	}
	if r.Count() != 2 {
		t.Errorf("count = %d", r.Count())
	}
	if u := r.Unresolved(); len(u) != 1 || u[0] != "ffffffffffffffffffffffffffffffff" {
		t.Errorf("unresolved = %v", u)
	}
}

func TestRefs(t *testing.T) {
	got := Refs("/_blob/" + id + " /_blob/" + id + " /_blob/short")
	if len(got) != 1 || got[0] != id {
		t.Errorf("refs = %v", got)
	}
}
