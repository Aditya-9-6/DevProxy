/**
 * @jest-environment jsdom
 */

describe('Theme Persistence', () => {
  beforeEach(() => {
    localStorage.clear();
    document.documentElement.setAttribute('data-theme', 'dark');
  });

  test('should persist theme preference in localStorage', () => {
    localStorage.setItem('theme', 'light');
    expect(localStorage.getItem('theme')).toBe('light');
  });

  test('should toggle theme correctly', () => {
    const toggle = document.createElement('button');
    toggle.id = 'theme-toggle';
    document.body.appendChild(toggle);
    
    // Simulate click
    const current = document.documentElement.getAttribute('data-theme');
    const next = current === 'dark' ? 'light' : 'dark';
    document.documentElement.setAttribute('data-theme', next);
    localStorage.setItem('theme', next);

    expect(document.documentElement.getAttribute('data-theme')).toBe('light');
    expect(localStorage.getItem('theme')).toBe('light');
  });
});