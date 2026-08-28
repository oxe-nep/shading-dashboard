import type { Metadata } from 'next';
import Link from 'next/link';
import './globals.css';

export const metadata: Metadata = {
  title: 'Shading Dashboard',
  description: 'Manage access VLAN assignments on shading switches',
  icons: {
    icon: 'https://cdn.prod.website-files.com/5bc9fe82c6c2f54b071f0033/5bc9fe82c6c2f5193f1f01c5_Untitled-4.png',
  },
};

export default function RootLayout({ children }: { children: React.ReactNode }) {
  return (
    <html lang="en">
      <head>
        <link
          rel="stylesheet"
          href="https://cdnjs.cloudflare.com/ajax/libs/font-awesome/6.5.1/css/all.min.css"
        />
        <link
          href="https://fonts.googleapis.com/css2?family=Inter:wght@400;500;600;700&display=swap"
          rel="stylesheet"
        />
      </head>
      <body>
        <div className="container">
          <AppHeader />
          {children}
        </div>
      </body>
    </html>
  );
}

function AppHeader() {
  return (
    <header className="compact-header">
      <div className="header-brand">
        <img src="/nep-logo.svg" alt="NEP" className="nep-logo" />
        <h1>Shading Dashboard</h1>
      </div>
      <nav className="header-nav">
        <Link href="/" className="header-settings" title="Dashboard">
          <i className="fas fa-th-large" />
        </Link>
        <Link href="/groups" className="header-settings" title="Port groups">
          <i className="fas fa-object-group" />
        </Link>
        <Link href="/vlans" className="header-settings" title="VLANs">
          <i className="fas fa-list" />
        </Link>
        <Link href="/config" className="header-settings" title="Configuration">
          <i className="fas fa-cog" />
        </Link>
      </nav>
    </header>
  );
}
