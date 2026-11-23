package expand
import "testing"
func TestExpand(t *testing.T){
 e:=New(Config{},MapLookup(map[string]string{"NAME":"lumet","EMPTY":""}))
 cases:=map[string]string{"$NAME":"lumet","$"+"{NAME}":"lumet","$"+"{MISSING:-fallback}":"fallback","$"+"{NAME:+yes}":"yes","$$NAME":"$NAME","\\$NAME":"$NAME"}
 for in,want:=range cases{got,err:=e.Expand(in);if err!=nil||got!=want{t.Errorf("%q => %q, %v; want %q",in,got,err,want)}}
}
func TestBacktickEscape(t *testing.T){e:=New(Config{Escape:96},MapLookup(map[string]string{"N":"x"}));got,err:=e.Expand(string(rune(96))+"$N");if err!=nil||got!="$N"{t.Fatalf("%q %v",got,err)}}
func TestStrictUnset(t *testing.T){if _,err:=New(Config{Strict:true},nil).Expand("$MISSING");err==nil{t.Fatal("expected error")}}
