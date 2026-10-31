import {test,expect} from '@playwright/test';
import {register} from './helpers';
import {mkdir} from 'node:fs/promises';
test('private upload persists after refresh and downloads exact original; logout and login',async({page},info)=>{
 const errors:string[]=[];page.on('pageerror',e=>errors.push(e.message));const email=await register(page);
 const body='Synthetic project brief\nGoal: make document handling traceable.\nBudget: 42 credits.\n';
 await page.getByLabel('Choose document').setInputFiles({name:'project-brief.txt',mimeType:'text/plain',buffer:Buffer.from(body)});
 await page.getByRole('button',{name:'Upload privately'}).click();await expect(page.getByRole('status').filter({hasText:'was saved privately and queued'})).toBeVisible();
 await expect(page.getByText('Queued',{exact:true})).toHaveCount(1);
 await page.reload();await expect(page.getByRole('heading',{name:'Welcome back'})).toHaveCount(0);await expect(page.getByRole('strong').filter({hasText:'project-brief.txt'})).toBeVisible();
 const downloadPromise=page.waitForEvent('download');await page.getByRole('link',{name:'Download project-brief.txt'}).click();const download=await downloadPromise;
 expect(download.suggestedFilename()).toBe('project-brief.txt');const stream=await download.createReadStream();let downloaded='';for await(const chunk of stream)downloaded+=chunk.toString();expect(downloaded).toBe(body);
 expect(await page.evaluate(()=>document.documentElement.scrollWidth<=innerWidth)).toBe(true);
 await mkdir('docs/screenshots',{recursive:true});await page.screenshot({path:`docs/screenshots/upload-queued-${info.project.name}.png`,fullPage:true});
 await page.getByRole('button',{name:'Sign out',exact:true}).click();await expect(page.getByRole('heading',{name:'Welcome back'})).toBeVisible();await expect(page.getByRole('heading',{name:'Local service health.'})).toHaveCount(0);await expect(page.getByLabel('Choose document')).toHaveCount(0);await expect(page.getByRole('strong').filter({hasText:'project-brief.txt'})).toHaveCount(0);
 await page.getByLabel('Email',{exact:true}).fill(email);await page.getByLabel('Password',{exact:true}).fill('Synthetic-password-123');await page.getByRole('button',{name:'Sign in',exact:true}).click();await expect(page.getByRole('strong').filter({hasText:'project-brief.txt'})).toBeVisible();
 // A rejected session must also discard a pending file before another sign-in.
 await page.getByLabel('Choose document').setInputFiles({name:'pending.txt',mimeType:'text/plain',buffer:Buffer.from('Synthetic unsent file')});
 await page.route('**/api/documents',route=>route.fulfill({status:401,contentType:'application/json',body:JSON.stringify({error:'authentication_required'})}));
 await page.getByRole('button',{name:'Refresh documents'}).click();await expect(page.getByRole('heading',{name:'Welcome back'})).toBeVisible();await page.unroute('**/api/documents');
 await page.getByLabel('Password',{exact:true}).fill('Synthetic-password-123');await page.getByRole('button',{name:'Sign in',exact:true}).click();
 await expect(page.getByRole('strong').filter({hasText:'project-brief.txt'})).toBeVisible();await expect(page.getByRole('button',{name:'Upload privately'})).toBeDisabled();await expect(page.getByLabel('Choose document')).toHaveValue('');
 expect(errors).toEqual([]);
});
test('invalid input and lost acknowledgement recover without duplicate documents',async({page},info)=>{
 await register(page);
 await page.getByLabel('Choose document').setInputFiles({name:'scan.pdf',mimeType:'application/pdf',buffer:Buffer.from('%PDF-1.7 synthetic')});await page.getByRole('button',{name:'Upload privately'}).click();await expect(page.getByRole('alert')).toContainText('PDF support is coming later');
 await page.screenshot({path:`docs/screenshots/upload-rejected-${info.project.name}.png`,fullPage:true});
 await page.getByLabel('Choose document').setInputFiles({name:'recovery.txt',mimeType:'text/plain',buffer:Buffer.from('Synthetic durable upload.')} );
 let firstKey='';await page.route('**/api/documents',async route=>{
  if(route.request().method()!=='POST'){await route.continue();return}
  firstKey=route.request().headers()['idempotency-key'];const response=await route.fetch();expect(response.status()).toBe(202);
  await route.fulfill({status:503,contentType:'application/json',body:JSON.stringify({error:'registration_unavailable'})});
 });
 await page.getByRole('button',{name:'Upload privately'}).click();await expect(page.getByRole('alert')).toContainText('Try again with the same file');await page.unroute('**/api/documents');
 const retry=page.waitForRequest(r=>r.method()==='POST'&&r.url().endsWith('/api/documents'));await page.getByRole('button',{name:'Upload privately'}).click();expect((await retry).headers()['idempotency-key']).toBe(firstKey);
 await expect(page.getByRole('strong').filter({hasText:'recovery.txt'})).toHaveCount(1);await expect(page.getByRole('alert')).toHaveCount(0);await page.reload();await expect(page.getByRole('heading',{name:'Welcome back'})).toHaveCount(0);await expect(page.getByRole('strong').filter({hasText:'recovery.txt'})).toHaveCount(1);await expect(page.getByText('Queued',{exact:true})).toHaveCount(1);
});
