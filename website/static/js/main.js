// ═══════════════════════════════════════════════════════════
//  gflow — Brutalist Interactions
//  Simple, clean animations for the light theme
// ═══════════════════════════════════════════════════════════

// ── Terminal Animation ───────────────────────────────────────
document.addEventListener('DOMContentLoaded', () => {
  const terminals = document.querySelectorAll('.terminal');
  
  terminals.forEach(terminal => {
    const lines = terminal.querySelectorAll('.line');
    lines.forEach((line, i) => {
      line.style.animationDelay = `${i * 0.15}s`;
    });
  });
});

// ── Smooth Scroll ────────────────────────────────────────────
document.querySelectorAll('a[href^="#"]').forEach(anchor => {
  anchor.addEventListener('click', function (e) {
    const href = this.getAttribute('href');
    if (href === '#') return;
    
    e.preventDefault();
    const target = document.querySelector(href);
    if (target) {
      target.scrollIntoView({
        behavior: 'smooth',
        block: 'start'
      });
    }
  });
});

// ── Card Hover Effects ───────────────────────────────────────
const cards = document.querySelectorAll('.bento-card, .feature-card, .cmd-card');

cards.forEach(card => {
  card.addEventListener('mouseenter', function() {
    this.style.transition = 'all 0.3s cubic-bezier(0.16, 1, 0.3, 1)';
  });
});
