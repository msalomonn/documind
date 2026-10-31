import {useEffect,useRef,useState} from 'react';
import {createRoot} from 'react-dom/client';
import './styles.css';
import {Workspace} from './workspace';
type Health={status:'ready'|'unavailable';components:{database:string;storage:string};documentProcessing:false};
function App(){
 const [health,setHealth]=useState<Health|null>(null),[loading,setLoading]=useState(true),[error,setError]=useState('');
 const active=useRef<AbortController|null>(null);
 async function refresh(){
  active.current?.abort();const controller=new AbortController();active.current=controller;
  setLoading(true);setError('');setHealth(null);
  const timer=setTimeout(()=>controller.abort('timeout'),8000);
  try{
   const response=await fetch('/api/health/ready',{signal:controller.signal});
   const data=await response.json();
   if(active.current!==controller)return;
   if(![200,503].includes(response.status)||!data.components)throw new Error('unavailable');
   setHealth(data);if(response.status!==200)setError('A local service is unavailable. Start the services, then try again.');
  }catch{if(active.current===controller)setError('Cannot reach your local workspace. Start the API, then try again.');}
  finally{clearTimeout(timer);if(active.current===controller)setLoading(false);}
 }
 useEffect(()=>{void refresh();return()=>{active.current?.abort();active.current=null;};},[]);
 return <div className="shell">
  <header className="topbar"><a className="brand" href="/" aria-label="DocuMind home"><span className="mark" aria-hidden="true">d<span>m</span></span>documind<span className="dot">.</span></a><span className="environment"><i/> LOCAL WORKSPACE</span></header>
  <main>
   <div className="intro"><div><p className="eyebrow">FROM DOCUMENTS TO UNDERSTANDING</p><h1>Make room<br/>for <em>clarity.</em></h1><p className="lede">A private place to turn your documents into<br className="desktop-break"/> structured insights, with a traceable path back to the source.</p></div><div className="paper-stack" aria-hidden="true"><div className="paper back"/><div className="paper front"><span className="paper-label">THE BIG PICTURE</span><div className="paper-line wide"/><div className="paper-line"/><div className="paper-line short"/><span className="highlight">Understand what matters.</span><div className="paper-line"/><div className="paper-line wide"/><span className="paper-number">01 /</span></div><span className="seal">dm.</span></div></div>
   <Workspace/>
   <section className="workspace" aria-labelledby="workspace-title"><div className="section-heading"><div><p className="eyebrow">YOUR STARTING POINT</p><h2 id="workspace-title">Local service health.</h2></div><span className="phase">Infrastructure</span></div>
    <div className="readiness"><div className="status-symbol" aria-hidden="true">{loading?'…':health?.status==='ready'?'✓':'!'}</div><div><h3>{loading?'Checking local services…':health?.status==='ready'?'Local services are ready':'Workspace needs attention'}</h3><p role="status">{loading?'Checking the database and private storage.':health?.status==='ready'?'Your database and private storage are connected.':'Some connections could not be verified.'}</p></div><button disabled={loading} onClick={()=>void refresh()}>{loading?'Checking…':'Check again'}<span aria-hidden="true"> ↗</span></button></div>
    {error&&<p className="error" role="alert">{error}</p>}
    <div className="services"><div><span className="service-icon" aria-hidden="true">▤</span><div><h3>Document records</h3><p>PostgreSQL</p></div><span className={'badge '+(health?.components.database==='ready'?'ok':'')}>{loading?'Checking':health?.components.database==='ready'?'Connected':'Unavailable'}</span></div><div><span className="service-icon" aria-hidden="true">◇</span><div><h3>Private file storage</h3><p>S3 compatible</p></div><span className={'badge '+(health?.components.storage==='ready'?'ok':'')}>{loading?'Checking':health?.components.storage==='ready'?'Connected':'Unavailable'}</span></div></div>

   </section>
   <footer><span>Built for understanding. Designed for trust.</span><span>Private originals · No AI calls</span></footer>
  </main>
 </div>;
}
createRoot(document.getElementById('root')!).render(<App/>);
