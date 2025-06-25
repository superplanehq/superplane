import React from 'react';
import ReactDOM from 'react-dom/client';
import './css/_app-defaults.css';
import './css/_app-fonts.css';
import './css/_app-buttons.css';
import './css/_app-forms.css';
import './css/_app-tabs.css';
import './css/_app-tables.css';
import './css/_app-badges.css';
import './css/_app-breadcrumbs.css';
import './css/_app-modals.css';
import './css/_custom-general.css';
import './css/_custom-dropdown.css';
import './css/_custom-main-menu.css';
import './css/_custom-pipeline.css';
import './css/_custom-structure.css';
import './css/_custom-loading-placeholders.css';
import './css/_custom-job.css';
import './css/_custom-insights.css';
import './css/_custom-billing.css';
import './css/_custom-datepicker.css';
import './css/_custom-help.css';
import './css/app-semaphore.css';
import './css/c3.min.css';
import './index.css';
import App from './App';
import reportWebVitals from './reportWebVitals';

// Suppress ResizeObserver loop error overlay in development
if (process.env.NODE_ENV === 'development') {
  const realConsoleError = console.error;
  console.error = (...args) => {
    if (
      typeof args[0] === 'string' &&
      args[0].includes('ResizeObserver loop completed with undelivered notifications')
    ) {
      // Ignore this error
      return;
    }
    realConsoleError(...args);
  };
}

const root = ReactDOM.createRoot(document.getElementById('root'));
root.render(
  <React.StrictMode>
    <App />
  </React.StrictMode>
);

// If you want to start measuring performance in your app, pass a function
// to log results (for example: reportWebVitals(console.log))
// or send to an analytics endpoint. Learn more: https://bit.ly/CRA-vitals
reportWebVitals();
