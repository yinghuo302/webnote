# webnote
Typora 是我们非常频繁使用的一款Markdown编辑器，它最棒的特性在于编辑时的即时渲染，但是缺点在于只有桌面版本，因此我们想要把它移植到网页端，适配以平板为代表的设备。以复现一个网页端的仿Typora编辑器为主体，同时通过后端的支持，实现云端储存、管理、编辑用户的Markdown笔记的功能。此外，向用户提供一个社区以分享自己的Markdown笔记，用户可以在任何设备任何场景访问或上传云端笔记，更加便捷高效；同时在社区里，用户可以展示自己的才华与成果，也可以与其他创作者交流互动，共同提高。

#### 实现效果

https://github.com/zanilia1016/webnote/assets/76104215/c142ba56-572c-447f-80ad-91c0e7781d56


![效果图](https://github.com/zanilia1016/webnote/assets/76104215/dcdce8b7-f772-4585-a08c-56a9746aaf18)


此仓库为webnote后端部分。主要使用Gin + Gorm 实现，实现的较为简单，只完成了基本功能部分。前端部分仓库在[webnote-web](https://github.com/zanilia1016/webnote-web)
