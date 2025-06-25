window.addEventListener("DOMContentLoaded", () => {
  const rulesBtn = document.getElementById('rulesBtn');
  const rulesModal = document.getElementById('rulesModal');
  const closeRules = document.getElementById('closeRules');
  
  function showModal() {
    rulesModal.classList.add('show');
    document.body.style.overflow = 'hidden';
  }
  
  function hideModal() {
    rulesModal.classList.remove('show');
    document.body.style.overflow = 'auto';
  }
  
  rulesBtn.addEventListener('click', showModal);
  closeRules.addEventListener('click', hideModal);
  rulesModal.addEventListener('click', (event) => {
    if (event.target === rulesModal) {
      hideModal();
    }
  });
  
  document.addEventListener('keydown', (event) => {
    if (event.key === 'Escape' && rulesModal.classList.contains('show')) {
      hideModal();
    }
  });
  
  const gameCard = document.querySelector('.game-card');
  let floatDirection = 1;
  
  setInterval(() => {
    const currentTransform = gameCard.style.transform || 'translateY(0px)';
    const currentY = parseFloat(currentTransform.match(/translateY\(([^)]+)\)/)?.[1] || 0);
    
    if (currentY >= 3) floatDirection = -1;
    if (currentY <= -3) floatDirection = 1;
    
    gameCard.style.transform = `translateY(${currentY + (floatDirection * 0.1)}px)`;
  }, 100);
})