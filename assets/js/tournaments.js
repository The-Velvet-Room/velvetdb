var select = document.getElementById("gametype")
if (select) {
  select.onchange = function() {
    select.disabled = true;
    window.location.href = "/tournaments/" + select.value;
  }
}
