import {defineConfig} from 'vite';
import react from '@vitejs/plugin-react';
export default defineConfig({root:'web',plugins:[react()],server:{host:'127.0.0.1',port:4320,strictPort:true,proxy:{'/api':{target:'http://127.0.0.1:4321'}}},build:{outDir:'../dist/web',emptyOutDir:true}});
