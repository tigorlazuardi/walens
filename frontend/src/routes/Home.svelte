<script>
  // Access runtime config injected by backend
  const config = window.__WALENS__ || {
    basePath: '/',
    apiBase: '/api',
  };
</script>

<div class="home">
  <h1>Walens</h1>
  <p>Welcome to Walens wallpaper manager</p>
  <p>Base path: {config.basePath}</p>
  <p>API base: {config.apiBase}</p>
</div>

<style>
  .home {
    text-align: center;
    padding: 2rem;
  }
</style>
