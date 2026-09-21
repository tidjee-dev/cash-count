import './app.css'
import { mount } from 'svelte'
import App from './App.svelte'
import { getLocaleSnapshot } from '$lib/i18n'

// Sync document language before first paint (screen readers, spellcheck).
if (typeof document !== 'undefined') {
  document.documentElement.lang = getLocaleSnapshot()
}

mount(App, { target: document.getElementById('app')! })
