import {expect,type Page} from '@playwright/test';
import {randomUUID} from 'node:crypto';
export async function register(page:Page){
 const email=`demo-${randomUUID()}@example.test`;
 await page.goto('/');await page.getByRole('button',{name:'New here? Create an account'}).click();
 await page.getByLabel('Email',{exact:true}).fill(email);await page.getByLabel('Password',{exact:true}).fill('Synthetic-password-123');await page.getByRole('button',{name:'Create account',exact:true}).click();
 await expect(page.getByText('Signed in as')).toContainText(email);await expect(page.getByRole('heading',{name:'Your library is ready for its first document.'})).toBeVisible();
 await expect(page.getByLabel('Password',{exact:true})).toHaveCount(0);
 return email;
}
