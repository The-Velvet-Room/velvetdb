$(function() {
  $('select').selectize({
    valueField: 'ID',
    labelField: 'Nickname',
    searchField: 'Nickname',
    onChange: function() {
      this.$wrapper.closest('.participant-row').find(':radio[value=select]').attr('checked', true);
    },
    load: function(query, callback) {
      if (!query.length) return callback();
      $.ajax({
        url: '/api/v1/players/search.json?query=' + encodeURIComponent(query),
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
