import assert from "node:assert/strict";
import { test } from "node:test";
import { stableChecks, validateStableEvidence } from "./stable-evidence.mjs";

const fixture = () => ({schemaVersion:1,version:"0.2.0",sourceRevision:"abc123",checks:Object.fromEntries(stableChecks.map(name=>[name,{passed:true,evidence:`Disposable test record for ${name}`,verifiedBy:"test fixture",verifiedOn:"2026-10-18"}]))});
test("stable promotion rejects absent, partial and failed evidence",()=>{
 assert.throws(()=>validateStableEvidence(null,"0.2.0","abc123"));
 for(const name of stableChecks){const value=fixture();delete value.checks[name];assert.throws(()=>validateStableEvidence(value,"0.2.0","abc123"),new RegExp(name));}
 const failed=fixture();failed.checks.gpu.passed=false;assert.throws(()=>validateStableEvidence(failed,"0.2.0","abc123"),/gpu/);
});
test("stable promotion ties reviewed records to the exact version and source",()=>{
 assert.throws(()=>validateStableEvidence(fixture(),"0.3.0","abc123"));
 assert.throws(()=>validateStableEvidence(fixture(),"0.2.0","new-source"));
 assert.doesNotThrow(()=>validateStableEvidence(fixture(),"0.2.0","abc123"));
 const incomplete=fixture();incomplete.checks["windows10-clean"].verifiedBy="";assert.throws(()=>validateStableEvidence(incomplete,"0.2.0","abc123"));
});
