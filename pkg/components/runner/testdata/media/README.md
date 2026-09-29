Regenerate with ffmpeg:

```
ffmpeg -f lavfi -i color=c=black:s=16x16:d=1 -an tiny.mp4
ffmpeg -f lavfi -i color=c=black:s=16x16:d=1 -c:v libvpx -an tiny.webm
ffmpeg -f lavfi -i color=c=black:s=16x16:d=1 -f lavfi -i anullsrc=r=16000:cl=mono -shortest silent.mp4
```

`malformed.mp4` is not a container. `misleading.mp4` is PNG bytes.
