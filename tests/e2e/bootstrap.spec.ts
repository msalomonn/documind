import {test,expect} from '@playwright/test';
import {mkdir} from 'node:fs/promises';
test('bootstrap shows real readiness without advertising working uploads',async({page},info)=>{
 const errors:string[]=[];page.on('pageerror',e=>errors.push(e.message));
 await page.goto('/');await expect(page.getByRole('heading',{name:'Local services are ready'})).toBeVisible();
 await expect(page.getByText('Connected',{exact:true})).toHaveCount(2);
 await expect(page.getByRole('heading',{name:'Document uploads are next.'})).toBeVisible();
 expect(await page.evaluate(()=>document.documentElement.scrollWidth<=innerWidth)).toBe(true);
 await mkdir('docs/screenshots',{recursive:true});await page.screenshot({path:`docs/screenshots/bootstrap-${info.project.name}.png`,fullPage:true});
 expect(errors).toEqual([]);
});
test('unavailable service shows an actionable error and recovers on retry',async({page},info)=>{
 await page.route('**/api/health/ready',route=>route.fulfill({status:503,contentType:'application/json',body:JSON.stringify({status:'unavailable',components:{database:'ready',storage:'unavailable'},documentProcessing:false})}));
 await page.goto('/');await expect(page.getByRole('alert')).toContainText('A local service is unavailable');
 await expect(page.getByText('Unavailable',{exact:true})).toHaveCount(1);
 await mkdir('docs/screenshots',{recursive:true});await page.screenshot({path:`docs/screenshots/bootstrap-unavailable-${info.project.name}.png`,fullPage:true});
 await page.unroute('**/api/health/ready');await page.getByRole('button',{name:'Check again'}).click();
 await expect(page.getByRole('heading',{name:'Local services are ready'})).toBeVisible();await expect(page.getByRole('alert')).toHaveCount(0);
});
