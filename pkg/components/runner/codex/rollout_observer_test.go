package codex

import (
	"encoding/json"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// The observer owns the persisted Codex protocol. These fixtures follow
// rust-v0.163.0 TokenUsageRecord, TurnCompleteEvent, and RateLimitSnapshot.
func TestRolloutObserverMeasuresOnlyCurrentResponses(t *testing.T) {
	script, err := filepath.Abs("rollout_observer.js")
	require.NoError(t, err)
	cmd := exec.Command("node", "-e", `
const fs = require("fs"), path = require("path");
const { createRolloutObserver } = require(process.argv[1]);
const home = process.argv[2], dir = path.join(home, "sessions", "2026", "10", "31");
fs.mkdirSync(dir, {recursive:true});
const file = path.join(dir, "rollout-2026-10-31T12-00-00-thread-1_reverted-1.jsonl");
const line = x => JSON.stringify(x) + "\n";
const usage = {input_tokens:100,cached_input_tokens:30,cache_write_input_tokens:10,output_tokens:20,reasoning_output_tokens:5,total_tokens:120};
const response = (id, value=usage) => ({type:"token_usage_record",payload:{thread_id:"thread-1",session_id:"root-1",turn_id:"turn-new",response_id:id,usage:value}});
fs.writeFileSync(file,line({type:"session_meta",payload:{id:"thread-1",session_id:"root-1"}})+line(response("historical")));
const snapshots=[];
const observer=createRolloutObserver({home,threadId:"thread-1",attemptId:"attempt-1",onSnapshot:s=>snapshots.push(s)});
fs.appendFileSync(file,line({type:"event_msg",payload:{type:"task_started",turn_id:"turn-new"}}));
const encoded=line(response("response-1"));
fs.appendFileSync(file,encoded.slice(0,-4));observer.poll();
const before=observer.snapshot();
fs.appendFileSync(file,encoded.slice(-4)+encoded);observer.poll();
const measured=observer.snapshot();
const invalid = [
 { ...usage, total_tokens: 999 },
 { ...usage, input_tokens: -1, cached_input_tokens: 0, cache_write_input_tokens: 0, total_tokens: 19 },
 { ...usage, input_tokens: Number.MAX_SAFE_INTEGER + 1, total_tokens: Number.MAX_SAFE_INTEGER + 21 },
 { ...usage, input_tokens: 1.5, cached_input_tokens: 0, cache_write_input_tokens: 0, total_tokens: 21.5 },
 { ...usage, input_tokens: "100" },
 { ...usage, cached_input_tokens: 101 },
 { ...usage, cache_write_input_tokens: -1 },
 { ...usage, reasoning_output_tokens: 21 },
 { input_tokens: Number.MAX_SAFE_INTEGER - 1, cached_input_tokens: 0, output_tokens: 1, reasoning_output_tokens: 0, total_tokens: Number.MAX_SAFE_INTEGER },
];
fs.appendFileSync(file,invalid.map((value,index)=>line(response("invalid-"+index,value))).join("")+line(response("response-1",{...usage,input_tokens:101,total_tokens:121})));
observer.poll();
const final=observer.finish({successful:false});
process.stdout.write(JSON.stringify({before,measured,final,snapshots}));
`, script, t.TempDir())
	out, err := cmd.CombinedOutput()
	require.NoError(t, err, string(out))
	var result map[string]any
	require.NoError(t, json.Unmarshal(out, &result))
	assert.Equal(t, "pending", result["before"].(map[string]any)["usage_state"])
	measured := result["measured"].(map[string]any)
	assert.Equal(t, "measured", measured["usage_state"])
	usage := measured["usage"].(map[string]any)
	assert.Equal(t, float64(60), usage["input_tokens"])
	assert.Equal(t, float64(15), usage["output_tokens"])
	assert.Equal(t, float64(120), usage["total_tokens"])
	final := result["final"].(map[string]any)
	assert.Equal(t, "partial", final["usage_state"])
	assert.Equal(t, usage, final["usage"])
}

func TestRolloutObserverCorrelatesStructuredQuotaEvidence(t *testing.T) {
	script, err := filepath.Abs("rollout_observer.js")
	require.NoError(t, err)
	cmd := exec.Command("node", "-e", `
const fs=require("fs"),path=require("path");
const {createRolloutObserver}=require(process.argv[1]);
const root=process.argv[2],results=[];
const entitlement="To use Codex with your ChatGPT plan, upgrade to Plus: https://chatgpt.com/explore/plus.";
for(const test of ["quota","entitlement","auth","rate-limit","http-429","context","unknown","stale","stale-terminal","ambiguous","workspace","api-key","wrong-session","wrong-payload-session","zero","missing","past-reset","invalid-reset","no-reset"]){
 const home=path.join(root,test),dir=path.join(home,"sessions");fs.mkdirSync(dir,{recursive:true});
 const observer=createRolloutObserver({home,attemptId:test,subscription:test!=="api-key"});
 const records=[{type:"session_meta",payload:{id:test==="wrong-session"?"unrelated":"thread-1",session_id:"root-1"}}];
 const event=payload=>records.push({type:"event_msg",payload});
 event({type:"task_started",turn_id:test==="stale"?"old":"new"});
 const reset = test === "past-reset" ? Math.floor(Date.now()/1000)-3600
   : test === "invalid-reset" ? "tomorrow" : test === "no-reset" ? undefined : Math.floor(Date.now()/1000)+3600;
 event({type:"token_count",rate_limits:{limit_id:"codex",primary:{used_percent:100,resets_at:reset},secondary:test==="ambiguous"?{used_percent:100,resets_at:reset+1}:null,rate_limit_reached_type:test==="workspace"?"workspace_owner_usage_limit_reached":"rate_limit_reached"}});
 if(test==="stale")event({type:"task_started",turn_id:"new"});
 if(test==="zero" || test==="wrong-payload-session")records.push({type:"token_usage_record",payload:{thread_id:"thread-1",session_id:test==="wrong-payload-session"?"unrelated":"root-1",turn_id:"new",response_id:"zero",usage:{input_tokens:0,cached_input_tokens:0,output_tokens:0,reasoning_output_tokens:0,total_tokens:0}}});
 const codes={auth:"unauthorized","rate-limit":"rate_limit_exceeded","http-429":{http_connection_failed:{http_status_code:429}},context:"context_window_exceeded",unknown:"other"};
 const successful=test==="zero" || test==="wrong-payload-session";
 event({type:"task_complete",turn_id:test==="stale-terminal"?"old":"new",error:successful?null:{codex_error_info:codes[test] || "usage_limit_exceeded",message:test==="entitlement"?entitlement:"A terminal provider error."}});
 fs.writeFileSync(path.join(dir,"rollout-test-thread-1.jsonl"),records.map(r=>JSON.stringify(r)).join("\n")+"\n");
 observer.setThreadId("thread-1");
 results.push({test,...observer.finish({successful})});
}
process.stdout.write(JSON.stringify(results));
`, script, t.TempDir())
	out, err := cmd.CombinedOutput()
	require.NoError(t, err, string(out))
	var results []map[string]any
	require.NoError(t, json.Unmarshal(out, &results))
	for _, result := range results {
		name := result["test"].(string)
		t.Run(name, func(t *testing.T) {
			switch name {
			case "entitlement", "auth", "rate-limit", "http-429", "context", "unknown", "stale-terminal", "api-key", "wrong-session", "wrong-payload-session", "zero":
				assert.Nil(t, result["failure"])
			default:
				failure := result["failure"].(map[string]any)
				assert.Equal(t, "chatgpt_quota_exhausted", failure["reason"])
				if name == "quota" {
					assert.NotEmpty(t, failure["resets_at"])
				} else if name == "stale" || name == "ambiguous" || name == "workspace" || name == "past-reset" || name == "invalid-reset" || name == "no-reset" {
					assert.Nil(t, failure["resets_at"])
				}
			}
			if name == "zero" {
				assert.Equal(t, "measured", result["usage_state"])
			}
			if name == "missing" || name == "wrong-payload-session" {
				assert.Equal(t, "unavailable", result["usage_state"])
			}
		})
	}
}
