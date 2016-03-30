# Gosser

The **Go** **S**even **S**egment Display Read**er**.

Takes one image (preferably in greyscale) and tries to convert it to numbers readable by a computer. This is performed by taking a central pixel as reference and looking north, south, west and east for segments shich should have a significantly darker (lower) value.

## Providing a source image
Using a image converting library such as imagemagick a source image can be enhanced to facilitate reading. Gosser works only with the red channel of the image, so converting the source image to greyscale is recommended. A minimum difference of 50% between active segment and reference pixel is needed. So increasing contrast is sometimes necessary.

This is an example of using imagemagick to crop and enhance a source image:

`convert source.jpg -rotate -5.8 -crop 100x22+311+206 -modulate 100,0 -level 25%,70% crop.jpg`

### Positioning 
`--positions` tells gosser how many digits should be located. Gosser assumes even spacing between digits. Gosser works best if there is a bit of room on each side of the digits.

Since no separators are detected, `--div` can be used to divide the result before output.

### Positioning using manifest file
You can tell gosser where to look for digits using a `manifest.json` file. This file holds simple x,y coordinates for the north and south reference pixel of each digit. Look at the example for details.

```
[{
    "north": [6, 7],
    "south": [6, 15]
}, {
    ...
}]
```

## Debugging the reader
Getting the reader to properly decode an image can be tricky if the image is small, fuzzy, or otherwise not tip top. There are some options available to help Gosser decode, such as the manifest file.

With the `--debug` option the inner workings of gosser can be examined. This can greatly help when the output is not what you expect.

If you also create a folder named `debug` gosser will write out the individual segments used in analysis. Overlaid are the search paths originating from the north and south reference pixels.

![0.png](https://github.com/dhogborg/gosser/blob/master/sample/debug/0.png?raw=true) ![1.png](https://github.com/dhogborg/gosser/blob/master/sample/debug/1.png?raw=true)![2.png](https://github.com/dhogborg/gosser/blob/master/sample/debug/2.png?raw=true)![3.png](https://github.com/dhogborg/gosser/blob/master/sample/debug/3.png?raw=true) ![4.png](https://github.com/dhogborg/gosser/blob/master/sample/debug/4.png?raw=true) ![5.png](https://github.com/dhogborg/gosser/blob/master/sample/debug/5.png?raw=true) ![6.png](https://github.com/dhogborg/gosser/blob/master/sample/debug/6.png?raw=true)

The std output will tell you which segments are considered active. In this example the south-west segment of the numer 2 has been missed, making the result unintelligible. In pedantic mode this will result in a error and no output.

```
*** *** *** ***   * *** *** 
* * * *   *   *   *   *   * 
* * * *   * ***   * *** *** 
* * * *   *       *   * *   
*** ***   * ***   * *** *** 
007-132
```


## Options

```
GLOBAL OPTIONS:
   --input, -i              input file
   --manifest, -m           Manifest file with coordinates for segments
   --positions, -p '0'      Number of digits in the image
   --output, -o 'string'    Output type, int or string
   --pedantic               Pedantic mode will output an error rather than let you see a invalid result
   --div '1'                Divide the result by a factor (only int output)
   --debug                  Enable debug output
   --help, -h               show help
   --version, -v            print the version
   ```