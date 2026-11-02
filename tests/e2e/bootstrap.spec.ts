import {test,expect} from '@playwright/test';
import {mkdir} from 'node:fs/promises';
import {register} from './helpers';
test('standalone login and registration keep application panels hidden',async({page},info)=>{
 const errors:string[]=[];page.on('pageerror',e=>errors.push(e.message));
 const privateRequests:string[]=[];page.on('request',r=>{if(r.url().includes('/api/documents')||r.url().includes('/api/health'))privateRequests.push(r.url())});
 await page.goto('/');await expect(page.getByRole('heading',{name:'Welcome back'})).toBeVisible();
 await expect(page.getByLabel('Choose document')).toHaveCount(0);await expect(page.getByRole('heading',{name:'Local service health.'})).toHaveCount(0);
 expect(await page.evaluate(()=>document.documentElement.scrollWidth<=innerWidth)).toBe(true);
 await mkdir('docs/screenshots',{recursive:true});await page.screenshot({path:`docs/screenshots/upload-signin-${info.project.name}.png`,fullPage:true});
 await page.getByRole('button',{name:'New here? Create an account'}).click();await expect(page.getByRole('heading',{name:'Create your account'})).toBeVisible();
 await page.screenshot({path:`docs/screenshots/upload-register-${info.project.name}.png`,fullPage:true});
 await page.getByRole('button',{name:'Already have an account? Sign in'}).click();
 await page.getByLabel('Email',{exact:true}).fill('missing@example.test');await page.getByLabel('Password',{exact:true}).fill('Wrong-password-123');await page.getByRole('button',{name:'Sign in',exact:true}).click();
 await expect(page.getByRole('alert')).toContainText('Email or password is incorrect');await expect(page.getByRole('heading',{name:'Welcome back'})).toBeVisible();expect(privateRequests).toEqual([]);expect(errors).toEqual([]);
});
test('session verification shows only loading until identity is resolved',async({page})=>{
 let release!:()=>void;const gate=new Promise<void>(r=>{release=r});
 await page.route('**/api/me',async route=>{await gate;await route.fulfill({status:401,contentType:'application/json',body:'{"error":"authentication_required"}'})});
 await page.goto('/');await expect(page.getByRole('status')).toHaveText('Checking your session…');
 await expect(page.getByLabel('Email',{exact:true})).toHaveCount(0);await expect(page.getByLabel('Choose document')).toHaveCount(0);await expect(page.getByRole('heading',{name:'Local service health.'})).toHaveCount(0);
 release();await expect(page.getByRole('heading',{name:'Welcome back'})).toBeVisible();
});
test('authenticated service health shows an actionable error and recovers',async({page},info)=>{
 await page.route('**/api/health/ready',route=>route.fulfill({status:503,contentType:'application/json',body:JSON.stringify({status:'unavailable',components:{database:'ready',storage:'unavailable'},documentProcessing:false})}));
 await register(page);await expect(page.getByRole('alert')).toContainText('A local service is unavailable');
 await expect(page.getByText('Unavailable',{exact:true})).toHaveCount(1);
 await mkdir('docs/screenshots',{recursive:true});await page.screenshot({path:`docs/screenshots/upload-unavailable-${info.project.name}.png`,fullPage:true});
 await page.unroute('**/api/health/ready');await page.getByRole('button',{name:'Check again'}).click();
 await expect(page.getByRole('heading',{name:'Local services are ready'})).toBeVisible();await expect(page.getByRole('alert')).toHaveCount(0);
});
