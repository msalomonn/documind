import {spawn} from 'node:child_process';
process.loadEnvFile('.env');
const args=process.argv.slice(2);
if(args[0]==='--integration'){process.env.DOCUMIND_INTEGRATION='1';args.shift();}
const child=spawn(args[0],args.slice(1),{stdio:'inherit',env:process.env});
for(const signal of ['SIGINT','SIGTERM'])process.on(signal,()=>child.kill(signal));
child.on('error',()=>{console.error('Command unavailable; check the README toolchain requirements.');process.exitCode=1;});
child.on('exit',(code,signal)=>{process.exitCode=code??(signal?1:0);});
