import {test,expect} from '@playwright/test';
import {execFileSync} from 'node:child_process';
import {register} from './helpers';
test('selected diagnostic job shows real success and failure after refresh',async({page},info)=>{
 await register(page);
 async function upload(name:string){await page.getByLabel('Choose document').setInputFiles({name,mimeType:'text/plain',buffer:Buffer.from('DOCUMIND WORKER SMOKE\nSynthetic browser fixture.\n')});const response=page.waitForResponse(r=>r.request().method()==='POST'&&r.url().endsWith('/api/documents'));await page.getByRole('button',{name:'Upload privately'}).click();const r=await response;expect(r.status()).toBe(202);return (await r.json()).id as string;}
 const id=await upload('worker-diagnostic.txt');
 execFileSync(process.execPath,['scripts/run.mjs','go','run','./cmd/worker','--smoke-document',id,'--once'],{timeout:30000,stdio:'pipe'});
 await page.getByRole('button',{name:'Refresh documents'}).click();await expect(page.getByText('Diagnostic complete',{exact:true})).toBeVisible();await expect(page.getByText(/Attempt 1\/3/)).toHaveCount(1);
 await page.screenshot({path:`docs/screenshots/worker-success-${info.project.name}.png`,fullPage:true});
 const failed=await upload('worker-failure.txt');
 execFileSync(process.execPath,['scripts/run.mjs','go','run','./cmd/worker','--smoke-document',failed,'--once','--smoke-permanent'],{timeout:30000,stdio:'pipe'});
 await page.getByRole('button',{name:'Refresh documents'}).click();await expect(page.getByText('Failed',{exact:true})).toBeVisible();await expect(page.getByText('smoke failure',{exact:true})).toBeVisible();await expect(page.getByText('Diagnostic complete',{exact:true})).toBeVisible();
 await page.screenshot({path:`docs/screenshots/worker-failure-${info.project.name}.png`,fullPage:true});
});
