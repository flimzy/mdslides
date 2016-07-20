importScripts('highlight.min.js');
onmessage = function(event) {
  var result = self.hljs.highlightAuto(event.data);
  postMessage(result.value);
}
