import {useEffect, useRef, useState, type FormEvent, type ReactNode} from 'react';
import {Brand} from './brand';
type User = {id:string;email:string};
type Document = {id:string;name:string;size:number;status:string;createdAt:string};
const messages:Record<string,string> = {
 authentication_required:'Your session has ended. Sign in again.', invalid_credentials:'Email or password is incorrect.',
 invalid_credentials_format:'Use a valid email and a password between 12 and 128 bytes.',account_unavailable:'An account with this email already exists. Try signing in.',
 auth_rate_limited:'Too many sign-in attempts. Wait a minute and try again.',pdf_not_enabled:'PDF support is coming later. Upload a UTF-8 .txt file for now.',
 invalid_text:'This file is empty or is not valid UTF-8 text.',text_too_long:'Use a document with no more than 200,000 characters.',file_too_large:'This file exceeds the 10 MiB limit.',
 unsupported_file:'Choose a UTF-8 .txt file.',idempotency_conflict:'This upload key was already used for different contents. Choose the file again.',
 storage_unavailable:'Private storage is unavailable. Your upload was not accepted. Try again.',registration_unavailable:'We could not confirm this upload. Try again with the same file to safely check its status.',
 uploads_busy:'Uploads are busy. Wait a moment and try again.',origin_rejected:'This address is not configured for sign-in. Use the local workspace address.',
};
export function Workspace({children}:{children:ReactNode}){
 const [user,setUser]=useState<User|null>(null),[checking,setChecking]=useState(true),[mode,setMode]=useState<'login'|'register'>('login');
 const [email,setEmail]=useState(''),[password,setPassword]=useState(''),[busy,setBusy]=useState(false),[error,setError]=useState(''),[notice,setNotice]=useState('');
 const [docs,setDocs]=useState<Document[]>([]),[listReady,setListReady]=useState(false),[file,setFile]=useState<File|null>(null);
 const key=useRef(''),input=useRef<HTMLInputElement>(null);
 async function api(path:string,init?:RequestInit){
  const response=await fetch(path,{...init,signal:AbortSignal.timeout(45000)});
  const data=await response.json();
  if(!response.ok){if(response.status===401&&path!=='/api/auth/login'){setUser(null);setDocs([]);setListReady(false);setFile(null);key.current='';setNotice('');setMode('login')};throw new Error(messages[data.error]??'Something went wrong. Check your local services and try again.');}
  return data;
 }
 useEffect(()=>{let active=true;void fetch('/api/me',{signal:AbortSignal.timeout(8000)}).then(async r=>{
  if(r.ok){const u=await r.json();if(active)setUser(u)}else if(r.status!==401)throw new Error();
 }).catch(()=>{if(active)setError('Cannot check your session. Check local services and try signing in.')}).finally(()=>{if(active)setChecking(false)});return()=>{active=false}},[]);
 useEffect(()=>{if(!user)return;let active=true;setListReady(false);
  void api('/api/documents').then(data=>{if(active){setDocs(data.documents);setListReady(true)}}).catch(e=>{if(active)setError(e.message)});return()=>{active=false};
 },[user]);
 async function signIn(event:FormEvent){event.preventDefault();setBusy(true);setError('');setNotice('');try{
  const u=await api('/api/auth/'+mode,{method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify({email,password})});setPassword('');setUser(u);
 }catch(e){setError(e instanceof Error?e.message:'Could not sign in.')}finally{setBusy(false)}}
 async function logout(){setBusy(true);setError('');try{await api('/api/auth/logout',{method:'POST'});setUser(null);setDocs([]);setListReady(false);setFile(null);key.current='';setNotice('');setMode('login')}catch(e){setError((e as Error).message)}finally{setBusy(false)}}
 async function upload(event:FormEvent){event.preventDefault();if(!file)return;setBusy(true);setError('');setNotice('');
  try{const form=new FormData();form.append('file',file);const d=await api('/api/documents',{method:'POST',headers:{'Idempotency-Key':key.current},body:form});
   setDocs(previous=>[d,...previous.filter(item=>item.id!==d.id)].slice(0,100));setListReady(true);setNotice(`${d.name} was saved privately and queued. Processing is not enabled yet.`);setFile(null);key.current='';if(input.current)input.current.value='';
  }catch(e){setError(e instanceof Error?e.message:'Could not confirm the upload. Retry with the same file.')}finally{setBusy(false)}
 }
 async function refresh(){setBusy(true);setError('');try{const d=await api('/api/documents');setDocs(d.documents);setListReady(true)}catch(e){setError((e as Error).message)}finally{setBusy(false)}}
 if(checking) return <div className="session-screen"><Brand/><main><div className="session-indicator" aria-hidden="true"/><p role="status">Checking your session…</p></main></div>;
 if(!user) return <div className="auth-screen">
  <aside className="auth-story"><Brand/><div className="auth-story-copy"><p className="eyebrow">FROM DOCUMENTS TO UNDERSTANDING</p><h2>Make room<br/>for <em>clarity.</em></h2><p>A private place for your documents.<br/>A clearer view of what matters.</p><div className="auth-paper" aria-hidden="true"><span className="paper-label">YOUR NEXT CHAPTER</span><div className="paper-line wide"/><div className="paper-line"/><span className="highlight">It starts with a little clarity.</span><div className="paper-line short"/><span className="paper-number">01 /</span></div></div><p className="auth-footnote">Built for understanding. Designed for trust.</p></aside>
  <main className="auth-main"><div className="auth-mobile-brand"><Brand/></div><div className="auth-form-wrap">
   <p className="eyebrow">YOUR PRIVATE WORKSPACE</p><h1>{mode==='register'?'Create your account':'Welcome back'}</h1><p className="auth-description">{mode==='register'?'A fresh space for your documents. Make it yours.':'Sign in to pick up where you left off.'}</p>
   <form className="identity-form" onSubmit={signIn}>
    <label htmlFor="email">Email</label><input id="email" type="email" placeholder="you@example.com" autoComplete="username" required maxLength={254} disabled={busy} value={email} onChange={e=>setEmail(e.target.value)}/>
    <label htmlFor="password">Password</label><input id="password" type="password" autoComplete={mode==='register'?'new-password':'current-password'} required minLength={12} maxLength={128} disabled={busy} value={password} onChange={e=>setPassword(e.target.value)}/>
    {mode==='register'&&<p className="hint">Use at least 12 characters.</p>}
    {error&&<p className="error" role="alert">{error}</p>}
    <button className="primary" disabled={busy}>{busy?'Please wait…':mode==='register'?'Create account':'Sign in'}</button>
    <button type="button" disabled={busy} className="text-button" onClick={()=>{setMode(mode==='register'?'login':'register');setError('');setPassword('')}}>{mode==='register'?'Already have an account? Sign in':'New here? Create an account'}</button>
   </form><p className="auth-privacy"><span aria-hidden="true">◇</span> Your documents are private to your account.</p>
  </div><p className="auth-local">LOCAL WORKSPACE</p></main>
 </div>;
 return <div className="shell application-shell">
  <header className="topbar"><Brand/><div className="account"><span>Signed in as <strong>{user.email}</strong></span><button disabled={busy} onClick={()=>void logout()}>Sign out</button></div></header>
  <main className="application-main"><section className="document-workspace" aria-labelledby="documents-title">
   <div className="section-heading"><div><p className="eyebrow">YOUR PRIVATE WORKSPACE</p><h1 id="documents-title">Your documents. One clear place.</h1></div></div>
   {error&&<p className="error" role="alert">{error}</p>}{notice&&<p className="success" role="status">{notice}</p>}
   <form className="upload-card" onSubmit={upload}><div className="upload-copy"><span className="upload-symbol" aria-hidden="true">↑</span><h3>Bring your first insight.</h3><p>Choose a UTF-8 text document. Only you can access the original.</p><p className="hint">.txt · Up to 10 MiB · 200,000 characters · PDF coming later</p></div><label className="file-label" htmlFor="document-file">Choose document</label><input ref={input} id="document-file" name="file" type="file" accept=".txt,text/plain" disabled={busy} onChange={e=>{setFile(e.target.files?.[0]??null);key.current=crypto.randomUUID();setError('');setNotice('')}}/><button className="primary" disabled={busy||!file} type="submit">{busy?'Please wait…':'Upload privately'}</button></form>
   <div className="list-heading"><h3>Recent documents <span className="count">{docs.length}</span></h3><button disabled={busy} onClick={()=>void refresh()}>Refresh documents</button></div>
   {docs.length===0?<div className="empty"><span aria-hidden="true">▤</span><h3>{listReady?'Your library is ready for its first document.':'Loading your library…'}</h3><p>{listReady?'Upload a text file to save a private original and register its processing job.':'Use Refresh documents if the connection fails.'}</p></div>:<ul className="document-list">{docs.map(d=><li key={d.id}><span className="document-icon" aria-hidden="true">▤</span><div className="document-name"><strong>{d.name}</strong><span>{d.size.toLocaleString()} bytes · {new Date(d.createdAt).toLocaleDateString()}</span></div><span className="badge queued">Queued</span><a className="download" href={`/api/documents/${d.id}/download`}>Download<span className="sr-only"> {d.name}</span></a></li>)}</ul>}
   <p className="pipeline-note">Saved and queued. Automatic processing and AI extraction are coming in the next stage; no AI calls are made.</p>

  </section>{children}<footer><span>Built for understanding. Designed for trust.</span><span>Private originals · No AI calls</span></footer></main>
 </div>;
}
