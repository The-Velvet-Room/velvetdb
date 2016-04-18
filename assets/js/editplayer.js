var select = document.getElementById("add-fact")
select.onclick = function() {
    document.getElementById("fact-list").innerHTML += '<div><input type="text" class="form-control" placeholder="Fact" name="facts"></div>'
}

select = document.getElementById("add-character")
select.onclick = function() {
    document.getElementById("character-list").innerHTML += '<div><input type="text" class="form-control" placeholder="Character" name="characters"></div>'
}

select = document.getElementById("add-alias")
select.onclick = function() {
    document.getElementById("alias-list").innerHTML += '<div><input type="text" class="form-control" placeholder="Alias" name="aliases"></div>'
}
