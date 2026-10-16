import type { ReactNode } from "react";
import wordmark from "@/assets/superplane-auth-wordmark.svg";
import { BrandArrowArt } from "@/components/branding/BrandArrowArt";
import "@/components/branding/brandSurface.css";
import "./LoginLayout.css";

export function LoginLayout({
  heading,
  subheading,
  children,
  footer,
}: {
  heading: string;
  subheading: string;
  children: ReactNode;
  footer?: ReactNode;
}) {
  return (
    <main className="brand-surface login-brand">
      <section className="login-brand-panel" aria-labelledby="login-heading">
        <a href="https://superplane.com" className="login-brand-home" aria-label="SuperPlane home">
          <img src={wordmark} alt="SuperPlane" width={2187} height={647} />
        </a>
        <div className="login-brand-form">
          <header>
            <h1 id="login-heading">{heading}</h1>
            <p className="login-brand-subheading">{subheading}</p>
          </header>
          <div className="login-brand-fields">{children}</div>
          {footer && <footer className="login-brand-terms">{footer}</footer>}
        </div>
        <p className="login-brand-caption">The open source software factory</p>
      </section>
      <aside className="login-brand-art" aria-label="SuperPlane">
        <div className="login-brand-message">
          <p className="login-brand-eyebrow">Built for engineering teams</p>
          <h2>
            Give your ideas
            <br />a way forward.
          </h2>
          <p>Turn your backlog into reviewed pull requests.</p>
        </div>
        <div className="login-brand-field">
          <BrandArrowArt />
        </div>
        <span className="login-brand-art-caption">From intent to implementation.</span>
      </aside>
    </main>
  );
}
