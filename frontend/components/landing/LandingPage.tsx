import Image from "next/image";
import Link from "next/link";
import { useEffect, useRef } from "react";
import {
  Activity,
  Boxes,
  ChartNoAxesCombined,
  CheckCircle2,
  Database,
  FileSearch,
  Gauge,
  HardDrive,
  LayoutGrid,
  ShieldCheck,
  Workflow,
  XCircle,
  Settings,
  Layers,
  Server,
  Shuffle,
  Play,
  Check,
  Zap,
  Terminal,
  Link2,
  Github,
  Cloud,
  Cpu
} from "lucide-react";

import styles from "./LandingPage.module.css";

type LandingPageProps = {
  authHref: string;
  authLabel: string;
};

const highlightData = [
  {
    icon: Zap,
    title: "No setup overhead",
    text: "Start monitoring in minutes, not hours.",
  },
  {
    icon: Boxes,
    title: "Docker-aware by default",
    text: "Works with your compose setup out of the box.",
  },
  {
    icon: Terminal,
    title: "Logs where you need them",
    text: "Click a container \u2192 see logs instantly.",
  },
  {
    icon: Server,
    title: "Built for small setups",
    text: "No enterprise complexity or heavy infra.",
  },
];

const useCases = [
  {
    title: "For Indie Hackers Running a Few VPS Servers",
    body: "Keep every machine, service, and container in one calm dashboard instead of spreading context across tabs and terminals.",
  },
  {
    title: "For Small SaaS Teams on Docker Compose",
    body: "Understand which project is unhealthy, which container restarted, and where logs are pointing you next.",
  },
  {
    title: "For Teams Without Dedicated DevOps",
    body: "Use a workflow that stays readable and operational without demanding observability-stack expertise first.",
  },
  {
    title: "For Self-Hosted Products",
    body: "Follow resource usage, service grouping, and container health with a setup that respects small-team constraints.",
  },
  {
    title: "For Faster Incident Debugging",
    body: "Move from host metrics to service groups to live logs fast when a workload starts acting strangely.",
  },
  {
    title: "For Growing Multi-VPS Setups",
    body: "Start simple now, keep clear visibility as new servers and services are added, and upgrade later when needed.",
  },
];

const environmentCards = [
  { title: "Docker", body: "Container stats and runtime visibility" },
  { title: "Docker Compose", body: "Automatic service grouping" },
  { title: "Standalone Containers", body: "Standalone workloads stay visible too" },
  { title: "Systemd Agent", body: "Simple host-level install path" },
  { title: "Self-Hosted Dashboard", body: "Free forever starting point" },
  { title: "Cloud Dashboard", body: "Hosted convenience when you want it" },
];

export function LandingPage({ authHref, authLabel }: LandingPageProps) {
  const headerRef = useRef<HTMLElement>(null);
  const innerRef = useRef<HTMLDivElement>(null);
  const sentinelRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    const sentinel = sentinelRef.current;
    const header = headerRef.current;
    const inner = innerRef.current;
    if (!sentinel || !header || !inner) return;

    const observer = new IntersectionObserver(
      ([entry]) => {
        const isScrolled = !entry.isIntersecting;
        header.classList.toggle(styles.headerScrolled, isScrolled);
        inner.classList.toggle(styles.navInnerScrolled, isScrolled);
      },
      { threshold: 0 }
    );

    observer.observe(sentinel);
    return () => observer.disconnect();
  }, []);

  return (
    <div className={styles.landingPage}>
      {/* Sentinel for IntersectionObserver scroll detection */}
      <div ref={sentinelRef} style={{ position: "absolute", top: 0, height: 1, width: 1 }} aria-hidden="true" />

      <header ref={headerRef} className={styles.header}>
        <div ref={innerRef} className={styles.navInner}>
          {/* Logo */}
          <Link className="inline-flex items-center gap-2.5 text-[1.1rem] font-bold tracking-[-0.03em] shrink-0" href="/">
            <svg
              className="h-6 w-6 drop-shadow-[0_0_12px_var(--landing-accent-soft)]"
              viewBox="0 0 32 32"
              fill="none"
              xmlns="http://www.w3.org/2000/svg"
              aria-hidden="true"
            >
              <defs>
                <linearGradient id="logo-glow" x1="0" y1="0" x2="32" y2="32" gradientUnits="userSpaceOnUse">
                  <stop stopColor="var(--landing-accent-strong)" />
                  <stop offset="1" stopColor="var(--landing-accent)" />
                </linearGradient>
              </defs>
              <path d="M16 4.5L5.5 10.5L16 16.5L26.5 10.5L16 4.5Z" fill="url(#logo-glow)" />
              <path d="M5.5 16.5L16 22.5L26.5 16.5" stroke="url(#logo-glow)" strokeWidth="3.2" strokeLinecap="round" strokeLinejoin="round" />
              <path d="M5.5 22.5L16 28.5L26.5 22.5" stroke="url(#logo-glow)" strokeWidth="3.2" strokeLinecap="round" strokeLinejoin="round" opacity="0.4" />
            </svg>
            <span>Bifrost</span>
          </Link>

          {/* Center nav pill – glass effect */}
          <nav
            aria-label="Landing navigation"
            className="hidden absolute left-1/2 -translate-x-1/2 items-center gap-0.5 rounded-full bg-[hsla(0_0%_100%_/_0.04)] backdrop-blur-2xl px-2 py-1.5 shadow-[0_2px_20px_hsla(0_0%_0%_/_0.3),inset_0_1px_0_hsla(0_0%_100%_/_0.04)] md:flex"
          >
            <a className="rounded-full px-4 py-2 text-[0.85rem] text-white/60 transition-colors duration-150 hover:bg-white/[0.07] hover:text-white/90" href="#features">
              Features
            </a>
            <a className="rounded-full px-4 py-2 text-[0.85rem] text-white/60 transition-colors duration-150 hover:bg-white/[0.07] hover:text-white/90" href="#deployment">
              Deployment
            </a>
            <a className="rounded-full px-4 py-2 text-[0.85rem] text-white/60 transition-colors duration-150 hover:bg-white/[0.07] hover:text-white/90" href="#pricing">
              Pricing
            </a>
          </nav>

          {/* CTA – floats independently */}
          <Link
            className="inline-flex items-center justify-center shrink-0 rounded-full border border-white/[0.08] bg-white/[0.06] px-5 py-2.5 text-[0.9rem] font-medium text-white/90 transition-all duration-150 hover:bg-white/[0.12] hover:text-white"
            href={authHref}
          >
            {authLabel}
          </Link>
        </div>
      </header>

      <main>
        <section className="relative isolate px-0 pb-9 pt-[4.2rem] text-center">
          <div aria-hidden="true" className="pointer-events-none absolute inset-0 z-0 overflow-hidden">
            <div
              className="absolute left-1/2 top-12 h-[30rem] w-[min(1080px,96vw)] -translate-x-1/2 rounded-full blur-3xl"
              style={{
                background:
                  "radial-gradient(circle at 50% 35%, color-mix(in srgb, var(--landing-accent-strong) 18%, transparent), transparent 58%)",
              }}
            />
            <div
              className="absolute left-1/2 top-48 h-[22rem] w-[min(860px,82vw)] -translate-x-1/2 rounded-full blur-3xl"
              style={{
                background:
                  "radial-gradient(circle, color-mix(in srgb, var(--landing-accent) 16%, transparent), transparent 64%)",
              }}
            />
          </div>
          <div className="relative z-1 mx-auto w-[min(1440px,calc(100%-2rem))]">
            <div className="relative z-5 mx-auto w-full max-w-[720px] pt-20">
              <div className="mb-6 inline-flex items-center gap-2 rounded-full border border-white/8 bg-[color:var(--landing-surface)] px-5 py-2 text-[0.85rem] font-normal text-[color:var(--landing-text-soft)]">
                <span className="h-[0.45rem] w-[0.45rem] rounded-full bg-[var(--landing-accent)] shadow-[0_0_12px_var(--landing-accent-soft)]" />
                The Open Source, Lightweight VPS Observability Platform You Need.
              </div>
              <h1 className="mx-auto w-full max-w-[850px] text-[clamp(2.8rem,5.5vw,4.5rem)] leading-[1.12] font-semibold tracking-[-0.03em] text-white text-balance">
                Monitor your VPS and Docker services without the overhead
              </h1>
              <p className="mx-auto mt-6 w-full max-w-[600px] text-[1.15rem] leading-[1.6] font-normal text-[color:var(--landing-text-muted)]">
                Metrics, services, and logs in one place.<br />
                No complex setup. No tool hopping.
              </p>

              <div className="mx-auto mt-10 flex flex-col sm:flex-row items-center justify-center gap-4 relative z-5">
                <Link
                  className={styles.buttonPrimary}
                  style={{ fontSize: "1.05rem", padding: "0.85rem 1.8rem", width: "100%", maxWidth: "240px" }}
                  href={authHref}
                >
                  Get Started Free
                </Link>
                <a
                  className={styles.buttonSecondary}
                  style={{ fontSize: "1.05rem", padding: "0.85rem 1.8rem", width: "100%", maxWidth: "240px" }}
                  href="https://github.com/dipeshsingh253/bifrost"
                  target="_blank"
                  rel="noopener noreferrer"
                >
                  View on GitHub
                </a>
              </div>

              <p className="mt-5 text-sm text-[color:var(--landing-text-soft)] text-center font-medium">
                Free &amp; open source &bull; Self-host or use cloud
              </p>
            </div>

            <div className="relative z-3 mt-12 flex justify-center">
              <div
                className="inline-block rounded-[1.8rem] p-[4px]"
                style={{
                  background:
                    "linear-gradient(180deg, color-mix(in srgb, var(--landing-accent-strong) 68%, transparent), color-mix(in srgb, var(--landing-accent) 22%, transparent) 24%, hsla(0 0% 100% / 0.05) 100%)",
                  boxShadow:
                    "0 30px 60px hsla(0 0% 0% / 0.6), 0 0 80px color-mix(in srgb, var(--landing-accent) 18%, transparent)",
                }}
              >
                <div className="relative overflow-hidden rounded-[calc(1.8rem-2px)] border border-white/10 bg-black">
                  <div className="relative flex justify-center">
                    <Image
                      alt="Bifrost dashboard showing server health and metrics"
                      className="h-auto max-w-full rounded-[inherit] object-top"
                      priority
                      src="/server-overview.png"
                      width={1889}
                      height={1049}
                      quality={100}
                      unoptimized
                    />
                  </div>
                </div>
              </div>
            </div>
          </div>
        </section>

        <section className={styles.section} id="features">
          <div className={styles.container}>
            <div className={styles.sectionHeader}>
              <h2 className={styles.sectionTitle} style={{ fontSize: "2.5rem" }}>Why Bifrost feels simple</h2>
              <p className={styles.sectionSubtitle} style={{ marginTop: "0.85rem", maxWidth: "600px", marginInline: "auto" }}>
                Everything you need to monitor, debug, and manage your infrastructure without the enterprise bloat.
              </p>
            </div>
            <div className={styles.problemRow}>
              {highlightData.map((item, idx) => {
                const Icon = item.icon;
                return (
                  <article key={idx} className={styles.problemCardItem}>
                    <div className={styles.problemCardIcon}>
                      <Icon size={20} />
                    </div>
                    <strong className={styles.problemCardTitle}>{item.title}</strong>
                    <span className={styles.problemCardText}>{item.text}</span>
                  </article>
                );
              })}
            </div>
          </div>
        </section>

        <section className={styles.zigZagSection} id="system">
          <div className={styles.container}>
            <div className={styles.zigZagHeader}>
              <h2 className={styles.sectionTitle}>Built for Operational Clarity</h2>
              <p className={styles.sectionSubtitle}>
                Get a complete overview of your infrastructure without drowning in dashboards.
              </p>
            </div>

            <div className={styles.zigZagRows}>
              <div className={styles.zigZagLine} />

              {/* Row 1 */}
              <div className={styles.zigZagRow}>
                <div className={styles.zigZagText}>
                  <h3 className={styles.zigZagTitle}>See all your servers in one place</h3>
                  <p className={styles.zigZagBody}>
                    Track CPU, memory, disk, network, and agent status across every connected server. Spot failing hosts instantly.
                  </p>
                </div>
                <div className={styles.zigZagVisual}>
                  <div className={styles.zigZagImageFrame}>
                    <Image
                      src="/all-servers.png"
                      alt="Bifrost Server List"
                      width={1919}
                      height={1079}
                      quality={100}
                      unoptimized
                      className={styles.zigZagImage}
                    />
                  </div>
                </div>
              </div>

              {/* Row 2 */}
              <div className={styles.zigZagRow}>
                <div className={styles.zigZagText}>
                  <h3 className={styles.zigZagTitle}>View metrics and containers together</h3>
                  <p className={styles.zigZagBody}>
                    Inspect resource usage with focused charts while viewing Docker Compose projects and standalone containers side-by-side.
                  </p>
                </div>
                <div className={styles.zigZagVisual}>
                  <div className={styles.zigZagImageFrame}>
                    <Image
                      src="/server-overview.png"
                      alt="Bifrost Server Dashboard"
                      width={1919}
                      height={1079}
                      quality={100}
                      unoptimized
                      style={{ objectPosition: 'center' }}
                      className={styles.zigZagImage}
                    />
                  </div>
                </div>
              </div>

              {/* Row 3 (New Differentiation Section) */}
              <div className={styles.zigZagRow}>
                <div className={styles.zigZagText}>
                  <h3 className={styles.zigZagTitle}>Understand your services, not just containers</h3>
                  <p className={styles.zigZagBody}>
                    Bifrost natively understands docker-compose groupings. Instead of a flat list of 50 containers, see your frontend, backend, and worker structure exactly as you defined them.
                  </p>
                </div>
                <div className={styles.zigZagVisual}>
                  <div className={styles.zigZagImageFrame}>
                    <Image
                      src="/server-docker-overview.png"
                      alt="Bifrost Compose Grouping"
                      width={1917}
                      height={924}
                      quality={100}
                      unoptimized
                      style={{ objectPosition: 'center bottom' }}
                      className={styles.zigZagImage}
                    />
                  </div>
                </div>
              </div>

              {/* Row 4 */}
              <div className={styles.zigZagRow}>
                <div className={styles.zigZagText}>
                  <h3 className={styles.zigZagTitle}>Live logs where you need them</h3>
                  <p className={styles.zigZagBody}>
                    Stop SSHing. Drill directly from a failing container state into its live log stream to debug issues immediately.
                  </p>
                </div>
                <div className={styles.zigZagVisual}>
                  <div className={styles.zigZagImageFrame}>
                    <Image
                      src="/container-logs.png"
                      alt="Bifrost Container Views"
                      width={990}
                      height={513}
                      quality={100}
                      unoptimized
                      className={styles.zigZagImage}
                    />
                  </div>
                </div>
              </div>

            </div>
          </div>
        </section>

        <section className={styles.section} id="deployment">
          <div className={styles.container}>
            <div className={styles.sectionHeader}>
              <span className={styles.pillLabel}>How It Works</span>
              <h2 className={styles.sectionTitle} style={{ marginTop: '1.25rem' }}>Get Started in 3 Easy Steps</h2>
              <p className={styles.sectionSubtitle}>
                Our agent is simple to install, easy to connect, and designed to give you server visibility right away.
              </p>
            </div>

            <div className={styles.stepGrid}>
              <article className={styles.stepCard}>
                <div className={styles.stepIconBlock}>
                  <Terminal size={24} />
                </div>
                <h3 className={styles.stepTitle}>1. Install</h3>
                <p className={styles.stepBody}>Run a single Docker command or install the systemd binary on your VPS.</p>
              </article>
              <article className={styles.stepCard}>
                <div className={styles.stepIconBlock}>
                  <Link2 size={24} />
                </div>
                <h3 className={styles.stepTitle}>2. Connect</h3>
                <p className={styles.stepBody}>
                  Provide your Bifrost dashboard URL and the unique agent token.
                </p>
              </article>
              <article className={styles.stepCard}>
                <div className={styles.stepIconBlock}>
                  <Activity size={24} />
                </div>
                <h3 className={styles.stepTitle}>3. Monitor</h3>
                <p className={styles.stepBody}>
                  Instantly see metrics, Docker Compose groups, and live container logs.
                </p>
              </article>
            </div>
          </div>
        </section>

        <section className={styles.section} id="open-source">
          <div className={styles.container}>
            <div className={styles.osGrid}>

              <div className={styles.osText}>
                <h2 className={styles.osTitle}>Open Source and Self-Hostable</h2>
                <p className={styles.osBody}>
                  We believe in transparency and control over your own data. Bifrost is fully open source. You can view our code on GitHub, contribute, or run it completely on your own hardware for free without any limitations.
                </p>
                <a className={styles.osLink} href="https://github.com/dipeshsingh253/bifrost" target="_blank" rel="noopener noreferrer">
                  View Source Code <span aria-hidden="true" style={{ marginLeft: "4px" }}>&rarr;</span>
                </a>
              </div>

              <div className={styles.osVisual}>
                <div className={styles.nodeGraph}>
                  <svg className={styles.nodeLines} viewBox="0 0 400 300" preserveAspectRatio="xMidYMid meet">
                    <path d="M 60 150 L 340 150" stroke="var(--landing-surface-border)" strokeWidth="2" fill="none" />
                    <path d="M 120 150 L 120 80 L 180 80" stroke="var(--landing-surface-border)" strokeWidth="2" fill="none" />
                    <path d="M 230 150 L 230 220 L 260 220" stroke="var(--landing-surface-border)" strokeWidth="2" fill="none" />
                    <path d="M 250 150 L 250 80 L 320 80" stroke="var(--landing-surface-border)" strokeWidth="2" fill="none" />
                  </svg>

                  <div className={styles.techNode} style={{ top: '50%', left: '15%' }}>
                    <Server size={22} color="var(--landing-text)" />
                  </div>
                  <div className={styles.techNode} style={{ top: '26.6%', left: '45%' }}>
                    <Boxes size={22} color="#0db7ed" />
                  </div>
                  <div className={styles.techNode} style={{ top: '50%', left: '50%' }}>
                    <Database size={22} color="#336791" />
                  </div>
                  <div className={styles.techNode} style={{ top: '73.3%', left: '65%' }}>
                    <Terminal size={22} color="var(--landing-text-muted)" />
                  </div>
                  <div className={styles.techNode} style={{ top: '26.6%', left: '80%' }}>
                    <LayoutGrid size={22} color="#f29400" />
                  </div>
                  <div className={styles.techNode} style={{ top: '50%', left: '85%' }}>
                    <Github size={22} color="#fff" />
                  </div>
                </div>
              </div>

            </div>
          </div>
        </section>

        <section className={styles.section} id="pricing">
          <div className={styles.container}>
            <div className={styles.sectionHeader}>
              <h2 className={styles.sectionTitle}>Simple Pricing for Growing Infrastructure</h2>
              <p className={styles.sectionSubtitle}>
                Start with self-hosted monitoring for free, then move to cloud when you want hosted convenience and
                longer retention.
              </p>
              <div className="mt-4 inline-flex items-center gap-2 rounded-full border border-white/10 bg-white/5 px-4 py-1.5 text-sm font-medium text-[var(--landing-accent)]">
                Free during beta. No credit card required.
              </div>
            </div>

            <div className={styles.pricingGrid}>
              <article className={styles.pricingCard}>
                <span className={styles.pricingEyebrow}>Self-Hosted</span>
                <h3 className={styles.cardTitle}>Free Forever</h3>
                <div className={styles.price}>
                  $0 <span>/ month</span>
                </div>
                <p className={styles.pricingBody}>
                  Run the dashboard and agent yourself with practical monitoring for personal infrastructure and small
                  teams.
                </p>
                <ul className={styles.featureList}>
                  <li>Unlimited servers</li>
                  <li>Unlimited services and containers</li>
                  <li>Basic monitoring</li>
                  <li>24h logs</li>
                  <li>Basic RBAC</li>
                </ul>
                <Link className={styles.buttonSecondary} href={authHref}>
                  Self-Host for Free
                </Link>
              </article>

              <article className={`${styles.pricingCard} ${styles.pricingFeatured}`}>
                <span className={styles.pricingEyebrow}>Cloud Starter</span>
                <h3 className={styles.cardTitle}>Hosted Convenience</h3>
                <div className={styles.price}>
                  $10 <span>/ server / month</span>
                </div>
                <p className={styles.pricingBody}>
                  Run only the agent while Bifrost hosts the dashboard, retention, and team-oriented basics for you.
                </p>
                <ul className={styles.featureList}>
                  <li>Hosted dashboard</li>
                  <li>Real-time monitoring</li>
                  <li>7-day logs</li>
                  <li>Basic alerts</li>
                  <li>Simple RBAC</li>
                </ul>
                <a className={styles.buttonPrimary} href="#use-cases">
                  Start on Cloud
                </a>
              </article>

              <article className={styles.pricingCard}>
                <span className={styles.pricingEyebrow}>Cloud Pro</span>
                <h3 className={styles.cardTitle}>For Growing Teams</h3>
                <div className={styles.price}>
                  $15 <span>/ server / month</span>
                </div>
                <p className={styles.pricingBody}>
                  Keep the same simple workflow while adding more retention, better team visibility, and alerting.
                </p>
                <ul className={styles.featureList}>
                  <li>Longer retention</li>
                  <li>Team features</li>
                  <li>Alerts</li>
                  <li>Better visibility at scale</li>
                  <li>Hosted operations simplicity</li>
                </ul>
                <a className={styles.buttonSecondary} href="#supported-environments">
                  Scale with Pro
                </a>
              </article>
            </div>
          </div>
        </section>



        {/* <section className={styles.section} id="testimonials">
          <div className={styles.container}>
            <div className={styles.sectionHeader}>
              <h2 className={styles.sectionTitle}>Love from developers</h2>
              <p className={styles.sectionSubtitle}>
                Join thousands of developers keeping their infrastructure reliable with Bifrost.
              </p>
            </div>

            <div className={styles.testimonialContainer}>
              <div className={styles.testimonialRow}>
                <article className={styles.testimonialCard}>
                  <h3 className={styles.testimonialHeadline}>Amazing clarity!</h3>
                  <p className={styles.testimonialBody}>
                    Finally, a monitoring tool that doesn't require a Ph.D. to set up. I connected my 5 VPS instances in under 10 minutes and had full metric visibility immediately.
                  </p>
                  <div className={styles.testimonialAuthor}>
                    <div className={styles.avatarPlaceholder} style={{ background: "linear-gradient(135deg, #FF6B6B, #FF8E53)" }}>AC</div>
                    <div className={styles.authorInfo}>
                      <strong>Alex Chen</strong>
                      <span>Backend Engineer</span>
                    </div>
                  </div>
                </article>
                <article className={styles.testimonialCard}>
                  <h3 className={styles.testimonialHeadline}>Perfect for lean teams</h3>
                  <p className={styles.testimonialBody}>
                    Just when I thought I couldn't find a lightweight alternative to Datadog, I found Bifrost. By grouping metrics around my Docker Compose services, it instantly made sense of my stack.
                  </p>
                  <div className={styles.testimonialAuthor}>
                    <div className={styles.avatarPlaceholder} style={{ background: "linear-gradient(135deg, #4D96FF, #6BCB77)" }}>SJ</div>
                    <div className={styles.authorInfo}>
                      <strong>Sarah Jenkins</strong>
                      <span>DevOps Consultant</span>
                    </div>
                  </div>
                </article>
                <article className={styles.testimonialCard}>
                  <h3 className={styles.testimonialHeadline}>Replaced my SSH habit</h3>
                  <p className={styles.testimonialBody}>
                    The built-in live log streaming is awesome. Not having to authenticate and SSH into individual servers just to see why a specific container crashed is a massive time saver.
                  </p>
                  <div className={styles.testimonialAuthor}>
                    <div className={styles.avatarPlaceholder} style={{ background: "linear-gradient(135deg, #9D4EDD, #C77DFF)" }}>MF</div>
                    <div className={styles.authorInfo}>
                      <strong>Michael Floyd</strong>
                      <span>Lead Developer</span>
                    </div>
                  </div>
                </article>
              </div>

              <div className={styles.testimonialRow} style={{ maxWidth: "850px" }}>
                <article className={styles.testimonialCard}>
                  <h3 className={styles.testimonialHeadline}>Actually open-source</h3>
                  <p className={styles.testimonialBody}>
                    After getting burned by &quot;open core&quot; pricing models that paywall basic features, Bifrost stands out. It&apos;s clean, lightweight, and wildly easy to throw onto a completely private homelab.
                  </p>
                  <div className={styles.testimonialAuthor}>
                    <div className={styles.avatarPlaceholder} style={{ background: "linear-gradient(135deg, #FFB703, #FB8500)" }}>DK</div>
                    <div className={styles.authorInfo}>
                      <strong>David Kim</strong>
                      <span>Software Engineer</span>
                    </div>
                  </div>
                </article>
                <article className={styles.testimonialCard}>
                  <h3 className={styles.testimonialHeadline}>Sleek and fast</h3>
                  <p className={styles.testimonialBody}>
                    The UI alone is arguably the best I&apos;ve seen in the open-source infrastructure space. It operates and feels exactly like a premium enterprise dashboard, but runs perfectly on a $5 VPS.
                  </p>
                  <div className={styles.testimonialAuthor}>
                    <div className={styles.avatarPlaceholder} style={{ background: "linear-gradient(135deg, #00B4D8, #0077B6)" }}>ER</div>
                    <div className={styles.authorInfo}>
                      <strong>Elena Rossi</strong>
                      <span>System Administrator</span>
                    </div>
                  </div>
                </article>
              </div>
            </div>
          </div>
        </section> */}

        <section className={styles.ctaSection}>
          <div className={styles.container}>
            <div className={styles.ctaPanel}>

              <svg className={styles.ctaSvgBorder} viewBox="0 0 100 100" preserveAspectRatio="none">
                <path d="M 0 35 L 15 35 L 25 70 L 35 70" fill="none" stroke="hsla(0 0% 100% / 0.15)" strokeWidth="0.5" vectorEffect="non-scaling-stroke" />
                <path d="M 100 35 L 85 35 L 75 70 L 65 70" fill="none" stroke="hsla(0 0% 100% / 0.15)" strokeWidth="0.5" vectorEffect="non-scaling-stroke" />
              </svg>

              <h2 className={styles.ctaTitle}>Start monitoring your servers in minutes</h2>
              <p className={styles.ctaSubtitle}>
                Join hundreds of teams monitoring their VPS, Docker containers, and logs effortlessly with Bifrost.
              </p>

              <Link className={styles.ctaButton} href={authHref}>
                Get Started Free
              </Link>

            </div>
          </div>
        </section>
      </main>

      <footer className={styles.footer}>
        <div className={styles.container}>
          <div className={styles.footerGrid}>
            <div className={styles.footerBrand}>
              <div className={styles.brand}>
                <svg
                  className="h-5 w-5 drop-shadow-[0_0_12px_var(--landing-accent-soft)]"
                  viewBox="0 0 32 32"
                  fill="none"
                  xmlns="http://www.w3.org/2000/svg"
                  aria-hidden="true"
                >
                  <path d="M16 4.5L5.5 10.5L16 16.5L26.5 10.5L16 4.5Z" fill="url(#logo-glow)" />
                  <path d="M5.5 16.5L16 22.5L26.5 16.5" stroke="url(#logo-glow)" strokeWidth="3.2" strokeLinecap="round" strokeLinejoin="round" />
                  <path d="M5.5 22.5L16 28.5L26.5 22.5" stroke="url(#logo-glow)" strokeWidth="3.2" strokeLinecap="round" strokeLinejoin="round" opacity="0.4" />
                </svg>
                <span>Bifrost</span>
              </div>
              <p>
                A simpler way to monitor VPS servers, Docker services, and logs for solo developers and growing small
                teams.
              </p>
            </div>

            <div className={styles.footerColumn}>
              <strong>Product</strong>
              <a href="#features">Features</a>
              <a href="#deployment">Deployment</a>
              <a href="#pricing">Pricing</a>
            </div>

            <div className={styles.footerColumn}>
              <strong>Workflows</strong>
              {/* <a href="#use-cases">Use Cases</a> */}
              {/* <a href="#supported-environments">Environments</a> */}
              <Link href="/setup">Admin Setup</Link>
            </div>

            <div className={styles.footerColumn}>
              <strong>Access</strong>
              <Link href="/login">Login</Link>
              <Link href={authHref}>{authLabel}</Link>
            </div>
          </div>

          <div className={styles.footerMeta}>
            <span>Built for self-hosted monitoring and practical cloud upgrades.</span>
            <span>Servers, services, containers, and logs in one focused dashboard.</span>
          </div>
        </div>
      </footer>
    </div>
  );
}
