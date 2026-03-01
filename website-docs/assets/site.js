(function () {
  var path = window.location.pathname.split('/').pop() || 'index.html';
  var links = document.querySelectorAll('[data-nav]');
  links.forEach(function (link) {
    if (link.getAttribute('href') === path) {
      link.classList.add('active');
    }
  });
  var year = document.getElementById('year');
  if (year) year.textContent = String(new Date().getFullYear());
})();
