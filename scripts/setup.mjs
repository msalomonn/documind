import {randomBytes} from 'node:crypto';
import {existsSync,writeFileSync} from 'node:fs';
const secret=()=>randomBytes(24).toString('hex');
if(!existsSync('.env')){
 const admin=secret(),app=secret(),s3admin=secret(),s3app=secret();
 writeFileSync('.env',`POSTGRES_PASSWORD=${admin}\nDOCUMIND_DB_PASSWORD=${app}\nDATABASE_ADMIN_URL=postgres://postgres:${admin}@127.0.0.1:55433/documind?sslmode=disable\nDATABASE_URL=postgres://documind_app:${app}@127.0.0.1:55433/documind?sslmode=disable\nS3_ENDPOINT=127.0.0.1:58333\nS3_BUCKET=documind-local\nS3_ADMIN_ACCESS_KEY=documind-bootstrap\nS3_ADMIN_SECRET_KEY=${s3admin}\nS3_ACCESS_KEY=documind-runtime\nS3_SECRET_KEY=${s3app}\nAPI_ADDRESS=127.0.0.1:4321\n`,{mode:0o600,flag:'wx'});
}
process.loadEnvFile('.env');
for(const key of ['S3_ADMIN_ACCESS_KEY','S3_ACCESS_KEY','S3_ADMIN_SECRET_KEY','S3_SECRET_KEY']){
 if(!/^[A-Za-z0-9-]+$/.test(process.env[key]??''))throw new Error('Invalid local storage credential format; inspect .env locally.');
}
if(process.env.S3_BUCKET!=='documind-local')throw new Error('Unsupported local bucket configuration.');
console.log('Local configuration ready; existing .env preserved.');
