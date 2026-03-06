/* ═══════════════════════════════════════════════════════════
   gflow — Physics-based Micro-Interactions Engine
   Spring dynamics, magnetic buttons, 3D tilt, parallax
   ═══════════════════════════════════════════════════════════ */

// ── Spring Physics Engine ────────────────────────────────────
class Spring {
  constructor(stiffness = 0.15, damping = 0.8) {
    this.stiffness = stiffness;
    this.damping = damping;
    this.value = 0;
    this.target = 0;
    this.velocity = 0;
  }
  update() {
    const force = (this.target - this.value) * this.stiffness;
    this.velocity = (this.velocity + force) * this.damping;
    this.value += this.velocity;
    return Math.abs(this.velocity) > 0.001 || Math.abs(this.target - this.value) > 0.001;
  }
  set(val) { this.target = val; }
  snap(val) { this.value = val; this.target = val; this.velocity = 0; }
}

// ── Cursor Glow — Follows mouse with spring physics ──────────
(function initCursorGlow() {
  const glow = document.createElement('div');
  glow.className = 'cursor-glow';
  document.body.appendChild(glow);

  const sx = new Spring(0.08, 0.82);
  const sy = new Spring(0.08, 0.82);
  let mouseX = window.innerWidth / 2;
  let mouseY = window.innerHeight / 2;
  let running = true;

  document.addEventListener('mousemove', e => {
    mouseX = e.clientX;
    mouseY = e.clientY;
  });

  function tick() {
    sx.set(mouseX);
    sy.set(mouseY);
    sx.update();
    sy.update();
    glow.style.transform = `translate(${sx.value - 300}px, ${sy.value - 300}px)`;
    if (running) requestAnimationFrame(tick);
  }
  tick();

  // Pause when tab not visible
  document.addEventListener('visibilitychange', () => {
    if (document.hidden) { running = false; }
    else { running = true; tick(); }
  });
})();

// ── Scroll Reveal — Staggered with IntersectionObserver ──────
const revealObserver = new IntersectionObserver((entries) => {
  entries.forEach(entry => {
    if (entry.isIntersecting) {
      entry.target.classList.add('visible');
      // Don't unobserve — allows re-entry effects if wanted
    }
  });
}, { threshold: 0.08, rootMargin: '0px 0px -60px 0px' });

document.querySelectorAll('.reveal').forEach(el => revealObserver.observe(el));

// Workflow steps get their own observer for sequential reveal
const stepObserver = new IntersectionObserver((entries) => {
  entries.forEach(entry => {
    if (entry.isIntersecting) {
      entry.target.classList.add('visible');
    }
  });
}, { threshold: 0.3 });

document.querySelectorAll('.workflow-step').forEach(el => stepObserver.observe(el));

// ── 3D Tilt — Feature cards with perspective transform ───────
(function initTiltCards() {
  const cards = document.querySelectorAll('.feature-card');

  cards.forEach(card => {
    // Add shine div
    const shine = document.createElement('div');
    shine.className = 'card-shine';
    card.appendChild(shine);

    const rx = new Spring(0.12, 0.75);
    const ry = new Spring(0.12, 0.75);
    let animating = false;

    function animate() {
      const moving = rx.update() | ry.update();
      card.style.transform = `perspective(800px) rotateX(${rx.value}deg) rotateY(${ry.value}deg) scale3d(1.02, 1.02, 1.02)`;
      if (moving) requestAnimationFrame(animate);
      else animating = false;
    }

    card.addEventListener('mousemove', e => {
      const rect = card.getBoundingClientRect();
      const x = (e.clientX - rect.left) / rect.width;
      const y = (e.clientY - rect.top) / rect.height;

      // Tilt: map 0-1 to -6 to 6 degrees
      rx.set((y - 0.5) * -12);
      ry.set((x - 0.5) * 12);

      // Shine position
      card.style.setProperty('--mouse-x', `${x * 100}%`);
      card.style.setProperty('--mouse-y', `${y * 100}%`);

      if (!animating) { animating = true; animate(); }
    });

    card.addEventListener('mouseleave', () => {
      rx.set(0);
      ry.set(0);
      if (!animating) { animating = true; animate(); }
      // Reset transform fully when spring settles
      setTimeout(() => {
        if (Math.abs(rx.value) < 0.01 && Math.abs(ry.value) < 0.01) {
          card.style.transform = '';
        }
      }, 500);
    });
  });
})();

// ── Terminal Typing Animation — Sequential reveal ────────────
(function initTerminalTyping() {
  const terminals = document.querySelectorAll('.terminal');

  const termObserver = new IntersectionObserver((entries) => {
    entries.forEach(entry => {
      if (entry.isIntersecting && !entry.target.dataset.animated) {
        entry.target.dataset.animated = 'true';
        const lines = entry.target.querySelectorAll('.line');
        lines.forEach((line, i) => {
          setTimeout(() => {
            line.classList.add('typed');
          }, 200 + i * 120);
        });
      }
    });
  }, { threshold: 0.3 });

  terminals.forEach(t => termObserver.observe(t));
})();

// ── Terminal 3D Tilt — Subtle perspective on hover ───────────
(function initTerminalTilt() {
  const terminals = document.querySelectorAll('.terminal');

  terminals.forEach(terminal => {
    const rx = new Spring(0.06, 0.8);
    const ry = new Spring(0.06, 0.8);
    let animating = false;

    function animate() {
      const moving = rx.update() | ry.update();
      terminal.style.transform = `perspective(1200px) rotateX(${rx.value}deg) rotateY(${ry.value}deg)`;
      if (moving) requestAnimationFrame(animate);
      else animating = false;
    }

    terminal.addEventListener('mousemove', e => {
      const rect = terminal.getBoundingClientRect();
      const x = (e.clientX - rect.left) / rect.width - 0.5;
      const y = (e.clientY - rect.top) / rect.height - 0.5;
      rx.set(y * -5);
      ry.set(x * 5);
      if (!animating) { animating = true; animate(); }
    });

    terminal.addEventListener('mouseleave', () => {
      rx.set(0);
      ry.set(0);
      if (!animating) { animating = true; animate(); }
    });
  });
})();

// ── Magnetic Buttons — Attracted to cursor ───────────────────
(function initMagneticButtons() {
  const buttons = document.querySelectorAll('.btn-primary, .btn-ghost, .btn-nav');

  buttons.forEach(btn => {
    const bx = new Spring(0.15, 0.7);
    const by = new Spring(0.15, 0.7);
    let animating = false;

    function animate() {
      const moving = bx.update() | by.update();
      btn.style.transform = `translate(${bx.value}px, ${by.value}px)`;
      if (moving) requestAnimationFrame(animate);
      else animating = false;
    }

    btn.addEventListener('mousemove', e => {
      const rect = btn.getBoundingClientRect();
      const cx = rect.left + rect.width / 2;
      const cy = rect.top + rect.height / 2;
      const dx = (e.clientX - cx) * 0.3;
      const dy = (e.clientY - cy) * 0.3;
      bx.set(dx);
      by.set(dy);
      if (!animating) { animating = true; animate(); }
    });

    btn.addEventListener('mouseleave', () => {
      bx.set(0);
      by.set(0);
      if (!animating) { animating = true; animate(); }
    });
  });
})();

// ── Copy Install Command — With haptic feedback ──────────────
document.querySelectorAll('.install-box').forEach(box => {
  box.addEventListener('click', () => {
    const cmd = box.querySelector('.cmd').textContent;
    navigator.clipboard.writeText(cmd).then(() => {
      const hint = box.querySelector('.copy-hint');
      if (hint) {
        const original = hint.textContent;
        hint.textContent = 'copied!';
        hint.style.color = '#34d399';

        // Micro-interaction: pulse the box
        box.style.transform = 'scale(0.97)';
        setTimeout(() => { box.style.transform = ''; }, 150);

        setTimeout(() => {
          hint.textContent = original;
          hint.style.color = '';
        }, 2000);
      }
    });
  });
});

// ── Nav — Scroll-aware with class toggle ─────────────────────
(function initNav() {
  const nav = document.querySelector('.nav');
  if (!nav) return;

  let lastScroll = 0;
  let ticking = false;

  window.addEventListener('scroll', () => {
    if (!ticking) {
      requestAnimationFrame(() => {
        const scrollY = window.scrollY;
        if (scrollY > 60) {
          nav.classList.add('scrolled');
        } else {
          nav.classList.remove('scrolled');
        }
        lastScroll = scrollY;
        ticking = false;
      });
      ticking = true;
    }
  });
})();

// ── Parallax — Hero elements with depth ──────────────────────
(function initParallax() {
  const hero = document.querySelector('.hero');
  if (!hero) return;

  const badge = hero.querySelector('.hero-badge');
  const h1 = hero.querySelector('h1');
  const sub = hero.querySelector('.hero-sub');
  const actions = hero.querySelector('.hero-actions');
  const install = hero.querySelector('.install-box');

  let ticking = false;

  window.addEventListener('scroll', () => {
    if (!ticking) {
      requestAnimationFrame(() => {
        const scrollY = window.scrollY;
        const rate = scrollY * 0.15;

        if (badge) badge.style.transform = `translateY(${rate * 0.5}px)`;
        if (h1) h1.style.transform = `translateY(${rate * 0.3}px)`;
        if (sub) sub.style.transform = `translateY(${rate * 0.4}px)`;
        if (actions) actions.style.transform = `translateY(${rate * 0.5}px)`;
        if (install) install.style.transform = `translateY(${rate * 0.6}px)`;

        // Fade hero as you scroll
        const opacity = Math.max(0, 1 - scrollY / 600);
        hero.style.opacity = opacity;

        ticking = false;
      });
      ticking = true;
    }
  });
})();

// ── Smooth Counter Animation — For stats ─────────────────────
function animateCounter(el, target, duration = 1500) {
  const start = performance.now();
  const initial = 0;

  function update(now) {
    const elapsed = now - start;
    const progress = Math.min(elapsed / duration, 1);
    // Ease out cubic
    const eased = 1 - Math.pow(1 - progress, 3);
    el.textContent = Math.round(initial + (target - initial) * eased);
    if (progress < 1) requestAnimationFrame(update);
  }
  requestAnimationFrame(update);
}

// ── Command Card Hover — Ripple effect ───────────────────────
document.querySelectorAll('.cmd-card').forEach(card => {
  card.addEventListener('mouseenter', e => {
    const rect = card.getBoundingClientRect();
    const x = e.clientX - rect.left;
    const y = e.clientY - rect.top;
    card.style.setProperty('--ripple-x', x + 'px');
    card.style.setProperty('--ripple-y', y + 'px');
  });
});
