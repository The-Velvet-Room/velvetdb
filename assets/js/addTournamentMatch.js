function checkPlayer(existing, playerString) {
  var value = existing ? 'select' : 'new';
  var inputs = document.getElementsByTagName('input');
  for (var index = 0; index < inputs.length; index++) {
    var element = inputs[index];
    if (element.type === 'radio' && element.name === playerString && element.value === value) {
      element.checked = true;
    }
  }
}

document.getElementsByName('player1newname')[0].onclick = function() {
  checkPlayer(false, 'player1');
}

document.getElementsByName('player2newname')[0].onclick = function() {
  checkPlayer(false, 'player2');
}

document.getElementsByName('player1select')[0].onchange = function() {
  checkPlayer(true, 'player1');
}

document.getElementsByName('player2select')[0].onchange = function() {
  checkPlayer(true, 'player2');
}
