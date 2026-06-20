import { StrictMode } from 'react'
import { createRoot } from 'react-dom/client'
import './index.css'
import ArticleListPage from './components/pages/ArticleListPage.tsx'

createRoot(document.getElementById('root')!).render(
  <StrictMode>
    <ArticleListPage />
  </StrictMode>,
)
