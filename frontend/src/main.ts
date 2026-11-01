import './style.css'
// Bootstrap modular border beam extensions
import './modules/bugu/beam'
import './modules/atproto/beam'

import App from './App.svelte'

const app = new App({
  target: document.getElementById('app')
})

export default app
