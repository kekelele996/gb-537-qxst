import { CssBaseline, ThemeProvider, createTheme } from '@mui/material'
import { StrictMode } from 'react'
import { createRoot } from 'react-dom/client'
import { App } from './App'
import './styles.css'

const theme = createTheme({
  palette: {
    mode: 'light',
    primary: { main: '#1d554f', dark: '#133d39', contrastText: '#f6faf7' },
    secondary: { main: '#556760' },
    error: { main: '#a43d39' },
    warning: { main: '#a56813' },
    success: { main: '#30745a' },
    background: { default: '#eef1ed', paper: '#f8faf7' },
    text: { primary: '#17221e', secondary: '#5d6964' },
    divider: '#cdd4cf',
  },
  shape: { borderRadius: 4 },
  typography: {
    fontFamily: '"Avenir Next", "Noto Sans SC", "PingFang SC", sans-serif',
    h1: { fontSize: '2rem', fontWeight: 700, lineHeight: 1.16, letterSpacing: 0 },
    h2: { fontSize: '1.35rem', fontWeight: 700, lineHeight: 1.25, letterSpacing: 0 },
    h3: { fontSize: '1rem', fontWeight: 700, lineHeight: 1.35, letterSpacing: 0 },
    button: { fontWeight: 700, textTransform: 'none', letterSpacing: 0 },
    body1: { letterSpacing: 0 }, body2: { letterSpacing: 0 }, caption: { letterSpacing: 0 },
  },
  components: {
    MuiButton: { styleOverrides: { root: { minHeight: 40, boxShadow: 'none' } } },
    MuiIconButton: { styleOverrides: { root: { borderRadius: 4 } } },
    MuiTextField: { defaultProps: { size: 'small' } },
    MuiFormControl: { defaultProps: { size: 'small' } },
    MuiTooltip: { defaultProps: { arrow: true } },
  },
})

createRoot(document.getElementById('root')!).render(<StrictMode><ThemeProvider theme={theme}><CssBaseline /><App /></ThemeProvider></StrictMode>)

