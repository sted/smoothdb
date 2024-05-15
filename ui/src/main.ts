import './app.css'
import App from './App.svelte'

export const adminDbUrl = import.meta.env.SMOOTHDB_URL + "admin";

const app = new App({
  target: document.getElementById('app')!,
})

export default app
