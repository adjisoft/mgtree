const root = document.documentElement;
const sidebar = document.getElementById('sidebar');
const sidebarToggle = document.getElementById('sidebar-toggle');
const themeToggle = document.getElementById('theme-toggle');
const mobileThemeToggle = document.getElementById('mobile-theme-toggle');
const themeLabel = document.getElementById('theme-label');
const navLinks = Array.from(document.querySelectorAll('.sidebar__nav a'));
const sections = navLinks
  .map((link) => document.querySelector(link.getAttribute('href')))
  .filter(Boolean);

const storageKey = 'mgtree-docs-theme';
const mediaQuery = window.matchMedia('(prefers-color-scheme: dark)');

function applyTheme(mode) {
  const effective = mode === 'auto' ? (mediaQuery.matches ? 'dark' : 'light') : mode;
  root.dataset.theme = effective;
  themeLabel.textContent = mode.charAt(0).toUpperCase() + mode.slice(1);
}

function cycleTheme() {
  const current = localStorage.getItem(storageKey) || 'auto';
  const next = current === 'auto' ? 'light' : current === 'light' ? 'dark' : 'auto';
  localStorage.setItem(storageKey, next);
  applyTheme(next);
}

function closeSidebar() {
  sidebar.classList.remove('is-open');
  if (sidebarToggle) {
    sidebarToggle.setAttribute('aria-expanded', 'false');
  }
}

function toggleSidebar() {
  const isOpen = sidebar.classList.toggle('is-open');
  if (sidebarToggle) {
    sidebarToggle.setAttribute('aria-expanded', String(isOpen));
  }
}

function setActiveLink() {
  let activeId = sections[0]?.id;
  const offset = window.scrollY + 160;
  for (const section of sections) {
    if (section.offsetTop <= offset) {
      activeId = section.id;
    }
  }
  navLinks.forEach((link) => {
    const isActive = link.getAttribute('href') === `#${activeId}`;
    link.classList.toggle('active', isActive);
  });
}

applyTheme(localStorage.getItem(storageKey) || 'auto');
mediaQuery.addEventListener('change', () => {
  if ((localStorage.getItem(storageKey) || 'auto') === 'auto') {
    applyTheme('auto');
  }
});

themeToggle?.addEventListener('click', cycleTheme);
mobileThemeToggle?.addEventListener('click', cycleTheme);
sidebarToggle?.addEventListener('click', toggleSidebar);
navLinks.forEach((link) => link.addEventListener('click', closeSidebar));
window.addEventListener('scroll', setActiveLink, { passive: true });
window.addEventListener('resize', () => {
  if (window.innerWidth > 1100) {
    closeSidebar();
  }
});
setActiveLink();
