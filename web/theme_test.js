describe('Theme Toggle', () => {
  test('should persist theme in localStorage', () => {
    localStorage.clear();
    document.documentElement.setAttribute('data-theme', 'dark');
    // Simulate toggle click
    const event = new Event('click');
    document.getElementById('theme-toggle').dispatchEvent(event);
    expect(localStorage.getItem('theme')).toBe('light');
  });
});