// Package expand provides a Dockerfile-oriented variable expander without shell parsing.
package expand
import("fmt";"strings")
type Lookup func(string)(string,bool)
type Config struct{Escape rune;PreserveUnset bool;Strict bool}
type Expander struct{config Config;lookup Lookup}
func New(c Config,l Lookup)*Expander{if c.Escape==0{c.Escape='\\'};if l==nil{l=func(string)(string,bool){return "",false}};return &Expander{c,l}}
func MapLookup(m map[string]string)Lookup{return func(k string)(string,bool){v,ok:=m[k];return v,ok}}
func(e *Expander)Expand(s string)(string,error){
 if e==nil{return s,nil};var b strings.Builder
 for i:=0;i<len(s);{
  if rune(s[i])==e.config.Escape&&i+1<len(s)&&s[i+1]=='$'{b.WriteByte('$');i+=2;continue}
  if s[i]!='$'{b.WriteByte(s[i]);i++;continue}
  if i+1<len(s)&&s[i+1]=='$'{b.WriteByte('$');i+=2;continue}
  if i+1<len(s)&&s[i+1]=='{'{end,err:=brace(s,i+2);if err!=nil{return "",err};v,err:=e.braced(s[i+2:end]);if err!=nil{return "",err};b.WriteString(v);i=end+1;continue}
  start,end:=i+1,i+1;for end<len(s)&&nameByte(s[end],end==start){end++};if end==start{b.WriteByte('$');i++;continue}
  name:=s[start:end];v,ok:=e.lookup(name);if !ok{if e.config.Strict{return "",fmt.Errorf("variable %q is not set",name)};if e.config.PreserveUnset{b.WriteString(s[i:end])}}else{b.WriteString(v)};i=end
 };return b.String(),nil
}
func(e *Expander)braced(s string)(string,error){
 name,op,arg:=split(s);if !valid(name){return "",fmt.Errorf("invalid variable expression: %s",s)}
 v,ok:=e.lookup(name);set:=ok&&v!=""
 switch op{
 case "":if !ok{if e.config.Strict{return "",fmt.Errorf("variable %q is not set",name)};if e.config.PreserveUnset{return "$"+"{"+s+"}",nil};return "",nil};return v,nil
 case "-",":-":if ok&&(op=="-"||set){return v,nil};return e.Expand(arg)
 case "+",":+":if ok&&(op=="+"||set){return e.Expand(arg)};return "",nil
 case "?",":?":if ok&&(op=="?"||set){return v,nil};if arg==""{arg=name+" is required"};msg,err:=e.Expand(arg);if err!=nil{return "",err};return "",fmt.Errorf("%s",msg)
 default:return "",fmt.Errorf("unsupported expansion operator %q",op)
 }
}
func split(s string)(string,string,string){for i:=0;i<len(s);i++{if s[i]==':'&&i+1<len(s)&&strings.ContainsRune("-+?",rune(s[i+1])){return s[:i],s[i:i+2],s[i+2:]};if strings.ContainsRune("-+?",rune(s[i])){return s[:i],s[i:i+1],s[i+1:]}};return s,"",""}
func brace(s string,start int)(int,error){n:=1;for i:=start;i<len(s);i++{if s[i]=='{'{n++};if s[i]=='}'{n--;if n==0{return i,nil}}};return 0,fmt.Errorf("unterminated variable expansion")}
func nameByte(c byte,first bool)bool{return c=='_'||c>='a'&&c<='z'||c>='A'&&c<='Z'||!first&&c>='0'&&c<='9'}
func valid(s string)bool{if s==""||!nameByte(s[0],true){return false};for i:=1;i<len(s);i++{if !nameByte(s[i],false){return false}};return true}
