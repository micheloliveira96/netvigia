package main

import (
 "context"
 "crypto/subtle"
 "encoding/json"
 "fmt"
 "log"
 "net"
 "net/http"
 "os"
 "os/exec"
 "path/filepath"
 "runtime"
 "strconv"
 "strings"
 "sync"
 "time"
)

type Config struct { Listen string `json:"listen"`; DataDir string `json:"data_dir"`; APIKey string `json:"api_key"`; PollMinutes int `json:"poll_minutes"`; SNMPCommunity string `json:"snmp_community"` }
type Device struct { ID string `json:"id"`; Name string `json:"name"`; Address string `json:"address"`; Type string `json:"type"`; Enabled bool `json:"enabled"`; IntervalMinutes int `json:"interval_minutes"`; SNMPVersion string `json:"snmp_version,omitempty"`; SNMPCommunity string `json:"snmp_community,omitempty"`; InterfaceIndex int `json:"interface_index,omitempty"`; LastStatus string `json:"last_status,omitempty"`; LastCheck time.Time `json:"last_check,omitempty"`; CreatedAt time.Time `json:"created_at"` }
type Metric struct { DeviceID string `json:"device_id"`; At time.Time `json:"at"`; Name string `json:"name"`; Value float64 `json:"value"`; Unit string `json:"unit,omitempty"` }
type State struct { Devices []Device `json:"devices"`; Metrics []Metric `json:"metrics"` }
type Server struct { mu sync.Mutex; cfg Config; state State; path string }

func main() {
 cfgPath:=os.Getenv("NETVIGIA_CONFIG"); if cfgPath=="" { cfgPath="/etc/netvigia/config.json" }
 cfg:=Config{Listen:"127.0.0.1:8787",DataDir:"/var/lib/netvigia",PollMinutes:60}
 if b,e:=os.ReadFile(cfgPath); e==nil { if e=json.Unmarshal(b,&cfg); e!=nil { log.Fatalf("configuração inválida: %v",e) } }
 if cfg.Listen=="" { cfg.Listen="127.0.0.1:8787" }; if cfg.DataDir=="" { cfg.DataDir="/var/lib/netvigia" }; if cfg.PollMinutes<1 { cfg.PollMinutes=60 }
 if e:=os.MkdirAll(cfg.DataDir,0750); e!=nil { log.Fatal(e) }
 s:=&Server{cfg:cfg,path:filepath.Join(cfg.DataDir,"state.json")}; s.load()
 mux:=http.NewServeMux(); mux.HandleFunc("/api/health",s.health); mux.HandleFunc("/api/devices",s.auth(s.devices)); mux.HandleFunc("/api/devices/",s.auth(s.deviceByID)); mux.HandleFunc("/api/metrics",s.auth(s.metrics)); mux.HandleFunc("/api/agent/metrics",s.auth(s.agentMetrics)); mux.HandleFunc("/api/check-now",s.auth(s.checkNow)); mux.Handle("/",s.web())
 go s.scheduler()
 log.Printf("NetVigia ouvindo em %s; intervalo padrão %d min",cfg.Listen,cfg.PollMinutes)
 log.Fatal(http.ListenAndServe(cfg.Listen,headers(mux)))
}
func (s *Server) load(){ b,e:=os.ReadFile(s.path); if e==nil { if json.Unmarshal(b,&s.state)!=nil { log.Printf("estado inválido; iniciando vazio"); s.state=State{} } } }
func (s *Server) save(){ b,e:=json.MarshalIndent(s.state,"","  "); if e!=nil { log.Printf("serialização: %v",e); return }; tmp:=s.path+".tmp"; if e=os.WriteFile(tmp,b,0640); e!=nil { log.Printf("gravação: %v",e); return }; if e=os.Rename(tmp,s.path); e!=nil { log.Printf("renomeação: %v",e) } }
func out(w http.ResponseWriter,c int,v interface{}){ w.Header().Set("Content-Type","application/json; charset=utf-8"); w.WriteHeader(c); _=json.NewEncoder(w).Encode(v) }
func (s *Server) health(w http.ResponseWriter,r *http.Request){ out(w,200,map[string]interface{}{"ok":true,"service":"NetVigia","version":"0.1.0","time":time.Now().UTC(),"platform":runtime.GOOS}) }
func (s *Server) auth(next http.HandlerFunc) http.HandlerFunc { return func(w http.ResponseWriter,r *http.Request){ if r.Method==http.MethodOptions { w.WriteHeader(204); return }; if s.cfg.APIKey!="" { got:=strings.TrimSpace(strings.TrimPrefix(r.Header.Get("Authorization"),"Bearer ")); if got=="" { got=r.Header.Get("X-API-Key") }; if subtle.ConstantTimeCompare([]byte(got),[]byte(s.cfg.APIKey))!=1 { out(w,401,map[string]string{"error":"não autorizado"}); return } }; next(w,r) } }
func (s *Server) devices(w http.ResponseWriter,r *http.Request){
 if r.Method==http.MethodGet { s.mu.Lock(); ds:=append([]Device(nil),s.state.Devices...); s.mu.Unlock(); out(w,200,ds); return }
 if r.Method!=http.MethodPost { w.Header().Set("Allow","GET, POST"); out(w,405,map[string]string{"error":"método não permitido"}); return }
 var d Device; if json.NewDecoder(http.MaxBytesReader(w,r.Body,1<<20)).Decode(&d)!=nil || strings.TrimSpace(d.Name)=="" || strings.TrimSpace(d.Address)=="" { out(w,400,map[string]string{"error":"informe nome e endereço/IP"}); return }
 d.ID=fmt.Sprintf("dev-%d",time.Now().UnixNano()); d.Name=strings.TrimSpace(d.Name); d.Address=strings.TrimSpace(d.Address); d.Type=strings.ToLower(strings.TrimSpace(d.Type)); if d.Type=="" { d.Type="generic" }; if d.IntervalMinutes<1 { d.IntervalMinutes=s.cfg.PollMinutes }; d.Enabled=true; d.CreatedAt=time.Now().UTC()
 s.mu.Lock(); s.state.Devices=append(s.state.Devices,d); s.save(); s.mu.Unlock(); out(w,201,d); go s.checkDevice(d)
}
func (s *Server) deviceByID(w http.ResponseWriter,r *http.Request){
 id:=strings.TrimPrefix(r.URL.Path,"/api/devices/"); s.mu.Lock(); defer s.mu.Unlock(); i:=-1; for n,d:=range s.state.Devices { if d.ID==id { i=n; break } }; if i<0 { out(w,404,map[string]string{"error":"equipamento não encontrado"}); return }
 switch r.Method { case http.MethodGet: out(w,200,s.state.Devices[i]); case http.MethodDelete: s.state.Devices=append(s.state.Devices[:i],s.state.Devices[i+1:]...); s.save(); w.WriteHeader(204); case http.MethodPatch: var p map[string]interface{}; if json.NewDecoder(r.Body).Decode(&p)!=nil { out(w,400,map[string]string{"error":"JSON inválido"}); return }; if v,ok:=p["enabled"].(bool); ok { s.state.Devices[i].Enabled=v }; if v,ok:=p["interval_minutes"].(float64); ok && v>=1 && v<=1440 { s.state.Devices[i].IntervalMinutes=int(v) }; s.save(); out(w,200,s.state.Devices[i]); default: w.Header().Set("Allow","GET, PATCH, DELETE"); out(w,405,map[string]string{"error":"método não permitido"}) }
}
func (s *Server) metrics(w http.ResponseWriter,r *http.Request){ if r.Method!=http.MethodGet { out(w,405,map[string]string{"error":"use GET"}); return }; id:=r.URL.Query().Get("device_id"); limit:=200; if n,e:=strconv.Atoi(r.URL.Query().Get("limit")); e==nil && n>0 && n<5000 { limit=n }; s.mu.Lock(); ms:=make([]Metric,0); for i:=len(s.state.Metrics)-1;i>=0 && len(ms)<limit;i-- { if id=="" || s.state.Metrics[i].DeviceID==id { ms=append(ms,s.state.Metrics[i]) } }; s.mu.Unlock(); out(w,200,ms) }
func (s *Server) agentMetrics(w http.ResponseWriter,r *http.Request){ if r.Method!=http.MethodPost { out(w,405,map[string]string{"error":"use POST"}); return }; var p struct { DeviceID string `json:"device_id"`; Name string `json:"name"`; Metrics map[string]float64 `json:"metrics"` }; if json.NewDecoder(http.MaxBytesReader(w,r.Body,1<<20)).Decode(&p)!=nil || p.DeviceID=="" || len(p.Metrics)==0 { out(w,400,map[string]string{"error":"device_id e metrics são obrigatórios"}); return }; now:=time.Now().UTC(); s.mu.Lock(); found:=false; for i:=range s.state.Devices { if s.state.Devices[i].ID==p.DeviceID { s.state.Devices[i].LastStatus="online"; s.state.Devices[i].LastCheck=now; if p.Name!="" { s.state.Devices[i].Name=p.Name }; found=true } }; if !found { s.mu.Unlock(); out(w,404,map[string]string{"error":"cadastre o equipamento primeiro"}); return }; for k,v:=range p.Metrics { s.state.Metrics=append(s.state.Metrics,Metric{DeviceID:p.DeviceID,At:now,Name:k,Value:v}) }; s.trim(); s.save(); s.mu.Unlock(); out(w,202,map[string]interface{}{"accepted":true,"received":len(p.Metrics)}) }
func (s *Server) checkNow(w http.ResponseWriter,r *http.Request){ if r.Method!=http.MethodPost { out(w,405,map[string]string{"error":"use POST"}); return }; go s.pollAll(); out(w,202,map[string]string{"status":"verificações iniciadas"}) }
func (s *Server) scheduler(){ s.pollAll(); t:=time.NewTicker(time.Minute); defer t.Stop(); for range t.C { s.pollAll() } }
func (s *Server) pollAll(){ s.mu.Lock(); ds:=append([]Device(nil),s.state.Devices...); s.mu.Unlock(); now:=time.Now(); for _,d:=range ds { if !d.Enabled { continue }; interval:=d.IntervalMinutes; if interval<1 { interval=s.cfg.PollMinutes }; if d.LastCheck.IsZero() || now.Sub(d.LastCheck)>=time.Duration(interval)*time.Minute { s.checkDevice(d) } } }
func (s *Server) checkDevice(d Device){ start:=time.Now(); ctx,cancel:=context.WithTimeout(context.Background(),3*time.Second); defer cancel(); err:=exec.CommandContext(ctx,"ping","-c","1","-W","2",d.Address).Run(); status:="offline"; if err==nil { status="online" }; now:=time.Now().UTC(); s.mu.Lock(); for i:=range s.state.Devices { if s.state.Devices[i].ID==d.ID { s.state.Devices[i].LastStatus=status; s.state.Devices[i].LastCheck=now; break } }; v:=0.0; if status=="online" { v=1 }; s.state.Metrics=append(s.state.Metrics,Metric{DeviceID:d.ID,At:now,Name:"availability",Value:v,Unit:"boolean"}); if status=="online" { s.state.Metrics=append(s.state.Metrics,Metric{DeviceID:d.ID,At:now,Name:"latency_ms",Value:float64(time.Since(start).Milliseconds()),Unit:"ms"}) }; s.trim(); s.save(); s.mu.Unlock(); if status=="online" && d.Type=="router" { s.snmpCheck(d) } }
func (s *Server) snmpCheck(d Device){ community:=d.SNMPCommunity; if community=="" { community=s.cfg.SNMPCommunity }; if community=="" { return }; version:=d.SNMPVersion; if version=="" { version="2c" }; get:=func(oid string)(float64,bool){ ctx,cancel:=context.WithTimeout(context.Background(),4*time.Second); defer cancel(); b,e:=exec.CommandContext(ctx,"snmpget","-v",version,"-c",community,"-Oqv",d.Address,oid).Output(); if e!=nil { return 0,false }; f,e:=strconv.ParseFloat(strings.Trim(strings.TrimSpace(string(b)),"\""),64); return f,e==nil }
 if d.InterfaceIndex>0 { for _,x:=range []struct{oid,name,unit string}{{fmt.Sprintf("IF-MIB::ifHCInOctets.%d",d.InterfaceIndex),"interface_in_octets","bytes"},{fmt.Sprintf("IF-MIB::ifHCOutOctets.%d",d.InterfaceIndex),"interface_out_octets","bytes"}} { if v,ok:=get(x.oid); ok { s.addCounterMetric(d.ID,x.name,v,x.unit) } } }
 if v,ok:=get("HOST-RESOURCES-MIB::hrSystemUptime.0"); ok { s.addMetric(d.ID,"uptime_ticks",v,"ticks") }
}
func (s *Server) addCounterMetric(id,n string,v float64,u string){ now:=time.Now().UTC(); rateName:=n+"_mbps"; s.mu.Lock(); var prev *Metric; for i:=len(s.state.Metrics)-1;i>=0;i-- { m:=s.state.Metrics[i]; if m.DeviceID==id && m.Name==n { cp:=m; prev=&cp; break } }; s.state.Metrics=append(s.state.Metrics,Metric{DeviceID:id,At:now,Name:n,Value:v,Unit:u}); if prev!=nil && v>=prev.Value { secs:=now.Sub(prev.At).Seconds(); if secs>0 { rate:=((v-prev.Value)*8/secs)/1000000; s.state.Metrics=append(s.state.Metrics,Metric{DeviceID:id,At:now,Name:rateName,Value:rate,Unit:"Mbps"}) } }; s.trim(); s.save(); s.mu.Unlock() }
func (s *Server) addMetric(id,n string,v float64,u string){ s.mu.Lock(); s.state.Metrics=append(s.state.Metrics,Metric{DeviceID:id,At:time.Now().UTC(),Name:n,Value:v,Unit:u}); s.trim(); s.save(); s.mu.Unlock() }
func (s *Server) trim(){ if len(s.state.Metrics)>200000 { s.state.Metrics=append([]Metric(nil),s.state.Metrics[len(s.state.Metrics)-150000:]...) } }
func (s *Server) web() http.Handler { root:=os.Getenv("NETVIGIA_WEB_DIR"); if root=="" { root="/opt/netvigia/web" }; if _,e:=os.Stat(filepath.Join(root,"index.html")); e!=nil { root="./web" }; return http.FileServer(http.Dir(root)) }
func headers(next http.Handler) http.Handler { return http.HandlerFunc(func(w http.ResponseWriter,r *http.Request){ w.Header().Set("X-Content-Type-Options","nosniff"); w.Header().Set("X-Frame-Options","DENY"); w.Header().Set("Referrer-Policy","strict-origin-when-cross-origin"); w.Header().Set("Access-Control-Allow-Origin","*"); w.Header().Set("Access-Control-Allow-Methods","GET, POST, PATCH, DELETE, OPTIONS"); w.Header().Set("Access-Control-Allow-Headers","Content-Type, Authorization, X-API-Key"); if r.Method==http.MethodOptions { w.WriteHeader(http.StatusNoContent); return }; next.ServeHTTP(w,r) }) }
var _ = net.ParseIP
