$(function() {
  $('select').selectize({
    valueField: 'URLPath',
    labelField: 'Nickname',
    searchField: 'Nickname',
    load: function(query, callback) {
      if (!query.length) return callback();
      $.ajax({
        url: '/api/v1/players/search?query=' + encodeURIComponent(query),
        type: 'GET',
        error: function() {
            callback();
        },
        success: function(res) {
            callback(res);
        }
      });
    }
  });
});
