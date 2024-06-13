import './app.css'
import { mount } from 'svelte';
import App from './App.svelte'

export const adminDbUrl = import.meta.env.SMOOTHDB_URL + "admin";

const app = mount(App, {target: document.getElementById('app')!})
export default app
