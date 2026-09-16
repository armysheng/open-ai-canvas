"""Render the original Chizao logomark as PNG. Requires Pillow."""
from pathlib import Path

from PIL import Image, ImageDraw

DEST = Path(__file__).resolve().parent


def render(foreground, name):
    scale = 8
    image = Image.new("RGBA", (128 * scale, 128 * scale))
    draw = ImageDraw.Draw(image)

    def polygon(points, fill=foreground):
        draw.polygon([(x * scale, y * scale) for x, y in points], fill=fill)

    # An open frame and a film strip form a geometric 片, with one emerging frame.
    polygon([(24, 19), (38, 19), (38, 55), (75, 55), (75, 69), (37, 69), (37, 87), (26, 109), (13, 102), (24, 81)])
    polygon([(57, 19), (71, 19), (71, 36), (97, 36), (97, 50), (57, 50)])
    polygon([(54, 78), (108, 78), (108, 109), (94, 109), (94, 92), (54, 92)])
    polygon([(92, 17), (107, 17), (107, 32), (92, 32)], "#E75B3D")
    image.resize((512, 512), Image.Resampling.LANCZOS).save(DEST / name)


if __name__ == "__main__":
    render("#222222", "logo-light.png")
    render("#F5F5F5", "logo-dark.png")
