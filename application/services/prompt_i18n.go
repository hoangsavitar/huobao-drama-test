package services

import (
	"fmt"

	"github.com/drama-generator/backend/pkg/config"
)

// PromptI18n is a prompt internationalization utility
type PromptI18n struct {
	config *config.Config
}

// NewPromptI18n creates a prompt internationalization utility
func NewPromptI18n(cfg *config.Config) *PromptI18n {
	return &PromptI18n{config: cfg}
}

// GetLanguage gets the current language setting
func (p *PromptI18n) GetLanguage() string {
	lang := p.config.App.Language
	if lang == "" {
		return "en" // default English
	}
	return lang
}

// IsEnglish checks if the current mode is English (dynamically reads config)
func (p *PromptI18n) IsEnglish() bool {
	return p.GetLanguage() == "en"
}

// GetStoryboardSystemPrompt gets the storyboard generation system prompt (English only).
func (p *PromptI18n) GetStoryboardSystemPrompt() string {
	return `[Role] You are a senior film storyboard artist, proficient in Robert McKee's shot breakdown theory, skilled at building emotional rhythm.

[Task] Break down the novel script into storyboard shots based on **independent action units**.

[Shot Breakdown Principles]
1. **Action Unit Division**: Each shot must correspond to a complete and independent action
   - One action = one shot (character stands up, walks over, speaks a line, reacts with an expression, etc.)
   - Do NOT merge multiple actions (standing up + walking over should be split into 2 shots)

2. **Shot Type Standards** (choose based on storytelling needs):
   - Extreme Long Shot (ELS): Environment, atmosphere building
   - Long Shot (LS): Full body action, spatial relationships
   - Medium Shot (MS): Interactive dialogue, emotional communication
   - Close-Up (CU): Detail display, emotional expression
   - Extreme Close-Up (ECU): Key props, intense emotions

3. **Camera Movement Requirements**:
   - Fixed Shot: Stable focus on one subject
   - Push In: Approaching subject, increasing tension
   - Pull Out: Expanding field of view, revealing context
   - Pan: Horizontal camera movement, spatial transitions
   - Follow: Following subject movement
   - Tracking: Linear movement with subject

4. **Emotion & Intensity Markers**:
   - Emotion: Brief description (excited, sad, nervous, happy, etc.)
   - Intensity: Emotion level using arrows
     * Extremely strong ↑↑↑ (3): Emotional peak, high tension
     * Strong ↑↑ (2): Significant emotional fluctuation
     * Moderate ↑ (1): Noticeable emotional change
     * Stable → (0): Emotion remains unchanged
     * Weak ↓ (-1): Emotion subsiding

[Output Requirements]
1. Generate an array, each element is a shot containing:
   - shot_number: Shot number
   - scene_description: Scene (location + time, e.g., "bedroom interior, morning")
   - shot_type: Shot type (extreme long shot/long shot/medium shot/close-up/extreme close-up)
   - camera_angle: Camera angle (eye-level/low-angle/high-angle/side/back)
   - camera_movement: Camera movement (fixed/push/pull/pan/follow/tracking)
   - action: Action description
   - result: Visual result of the action
   - dialogue: Character dialogue or narration (if any)
   - emotion: Current emotion
   - emotion_intensity: Emotion intensity level (3/2/1/0/-1)

**CRITICAL: Return ONLY a valid JSON array. Do NOT include any markdown code blocks, explanations, or other text. Start directly with [ and end with ].**

[Important Notes]
- Shot count must match number of independent actions in the script (not allowed to merge or reduce)
- Each shot must have clear action and result
- Shot types must match storytelling rhythm (don't use same shot type continuously)
- Emotion intensity must accurately reflect script atmosphere changes`
}

// GetSceneExtractionPrompt gets the scene extraction prompt (English only).
func (p *PromptI18n) GetSceneExtractionPrompt(style, aspectRatio string) string {
	if aspectRatio == "" {
		aspectRatio = "16:9"
	}
	imageRatio := aspectRatio

	return fmt.Sprintf(`[Task] Extract all unique scene backgrounds from the script

[Requirements]
1. Identify all different scenes (location + time combinations) in the script
2. Generate detailed **English** image generation prompts for each scene
3. **Important**: Scene descriptions must be **pure backgrounds** without any characters, people, or actions
4. Prompt requirements:
   - Must use **English**, no Chinese characters
   - Detailed description of scene, time, atmosphere, style
   - Must explicitly specify "no people, no characters, empty scene"
   - Must match the drama's genre and tone
   - **Style Requirement**: %s
   - **Image Ratio**: %s


[Output Format]
**CRITICAL: Return ONLY a valid JSON array. Do NOT include any markdown code blocks, explanations, or other text. Start directly with [ and end with ].**

Each element containing:
- location: Location (e.g., "luxurious office")
- time: Time period (e.g., "afternoon")
- prompt: Complete English image generation prompt (pure background, explicitly stating no people)`, p.styleLabel(style), imageRatio)
}

// GetFirstFramePrompt gets the first frame prompt (English only).
func (p *PromptI18n) GetFirstFramePrompt(style, aspectRatio string) string {
	if aspectRatio == "" {
		aspectRatio = "16:9"
	}
	imageRatio := aspectRatio
	return fmt.Sprintf(`You are a professional image generation prompt expert. Please generate prompts suitable for AI image generation based on the provided shot information.

Important: This is the first frame of the shot - a completely static image showing the initial state before the action begins.

Key Points:
1. Focus on the initial static state - the moment before the action
2. Must NOT include any action or movement
3. Describe the character's initial posture, position, and expression
4. Can include scene atmosphere and environmental details
5. Shot type determines composition and framing:
   - If "Close-up" or "Extreme Close-up": Must include "heavily blurred background of [Location], shallow depth of field, focus on face" to ensure spatial logic.
   - For multi-character interactive shots: Explicitly define L/R positioning (e.g., "Character A on the left, facing right towards Character B") to ensure correct reverse-angle logic in subsequent shots.
- **Style Requirement**: %s
- **Image Ratio**: %s
Output Format:
Return a JSON object containing only:
- prompt: Complete English image generation prompt (detailed description, suitable for AI image generation)`, p.styleLabel(style), imageRatio)
}

// GetKeyFramePrompt gets the key frame prompt (English only).
func (p *PromptI18n) GetKeyFramePrompt(style, aspectRatio string) string {
	if aspectRatio == "" {
		aspectRatio = "16:9"
	}
	imageRatio := aspectRatio
	return fmt.Sprintf(`You are a professional image generation prompt expert. Please generate prompts suitable for AI image generation based on the provided shot information.

Important: This is the key frame of the shot - capturing the most intense and exciting moment of the action.

Key Points:
1. Focus on the most exciting moment of the action
2. Capture peak emotional expression
3. Emphasize dynamic tension
4. Show character actions and expressions at their climax
5. Can include motion blur or dynamic effects
- **Style Requirement**: %s
- **Image Ratio**: %s
Output Format:
Return a JSON object containing only:
- prompt: Complete English image generation prompt (detailed description, suitable for AI image generation)`, p.styleLabel(style), imageRatio)
}

// GetActionSequenceFramePrompt gets the action sequence frame prompt (English only).
func (p *PromptI18n) GetActionSequenceFramePrompt(style, aspectRatio string) string {
	if aspectRatio == "" {
		aspectRatio = "16:9"
	}
	imageRatio := aspectRatio
	return fmt.Sprintf(`**Role:** You are an expert in visual storytelling and image generation prompting. You need to generate a single prompt that describes a 3x3 grid action sequence.

**Core Logic:**

1. **Holistic Integration:** This is a single, complete image containing a 3x3 grid layout, showcasing 9 sequential actions of the same subject.
2. **Visual Anchoring:** The subject, clothing, art style, and character consistency must be identical across all 9 frames.
3. **Action Evolution:** From Frame 1 to Frame 9, display a complete action sequence (e.g., Standing → Walking → Running → Jumping → Landing).
4. **Prompt Engineering:** Use high-quality visual vocabulary (lighting, textures, composition, depth of field).

**Important:**

You must generate **ONE** comprehensive prompt to describe the entire 3x3 grid image, rather than 9 independent prompts.

Each frame **must** follow these specific rules:

- **Frame 1:** Preparation/Initial stance
- **Frame 2:** Anticipation/Body adjustment
- **Frame 3:** Initiation/Beginning of movement
- **Frame 4:** Acceleration/Power building
- **Frame 5:** Peak of tension/Just before the burst
- **Frame 6:** Action burst/The climax moment
- **Frame 7:** Power release/Inertia continuation
- **Frame 8:** Deceleration/Follow-through
- **Frame 9:** Complete conclusion/Return to stillness

**Aspect Ratio:** * %s

**Output Specification:**

You must return a **JSON object** with only:

- **prompt**: A **complete English image generation prompt** (describing the 3x3 grid layout, subject features, the evolution of the 9 actions, environment, and lighting details to ensure the AI generates one single image containing 9 frames).

**Example Format:**

{
  "prompt": "Action sequence layout, 3x3 grid composition\n [Frame 1]: [Subject] standing naturally in [Setting], feet shoulder-width apart...\n---\n [Frame 2]: [Subject] locking eyes forward, leaning body slightly...\n---\n [Frame 3]: [Subject's legs] bending slightly, center of gravity lowering...\n---\n [Frame 4]: [Subject] pushing off with back leg, body moving forward, dust rising from [Setting's ground]...\n---\n [Frame 5]: [Subject's clothing] fluttering, body leaning deep, fist charging power...\n---\n [Frame 6]: [Subject] sprinting at full speed, fist striking out...\n---\n [Frame 7]: [Subject] impact moment, body lunging forward...\n---\n [Frame 8]: [Subject] slowing down, pulling back the fist...\n---\n [Frame 9]: [Subject's full appearance] standing firm in [Setting], recovering original stance."
}

`, p.styleLabel(style), imageRatio)
}

// GetLastFramePrompt gets the last frame prompt (English only).
func (p *PromptI18n) GetLastFramePrompt(style, aspectRatio string) string {
	if aspectRatio == "" {
		aspectRatio = "16:9"
	}
	imageRatio := aspectRatio
	return fmt.Sprintf(`You are a professional image generation prompt expert. Please generate prompts suitable for AI image generation based on the provided shot information.

Important: This is the last frame of the shot - a static image showing the final state and result after the action ends.

Key Points:
1. Focus on the final state after action completion
2. Show the result of the action
3. Describe character's final posture and expression after action
4. Emphasize emotional state after action
5. Capture the calm moment after action ends
- **Style Requirement**: %s
- **Image Ratio**: %s
Output Format:
Return a JSON object containing only:
- prompt: Complete English image generation prompt (detailed description, suitable for AI image generation)`, p.styleLabel(style), imageRatio)
}

// GetOutlineGenerationPrompt gets the outline generation prompt (English only).
func (p *PromptI18n) GetOutlineGenerationPrompt() string {
	return `You are a professional short drama screenwriter. Based on the theme and number of episodes, create a complete short drama outline and plan the plot direction for each episode.

Requirements:
1. Compact plot with strong conflicts and fast pace
2. Each episode should have independent conflicts while connecting the main storyline
3. Clear character arcs and growth
4. Cliffhanger endings to hook viewers
5. Clear theme and emotional core

Output Format:
Return a JSON object containing:
- title: Drama title (creative and attractive)
- episodes: Episode list, each containing:
  - episode_number: Episode number
  - title: Episode title
  - summary: Episode content summary (50-100 words)
  - conflict: Main conflict point
  - cliffhanger: Cliffhanger ending (if any)`
}

// GetCharacterExtractionPrompt gets the character extraction prompt (English only).
func (p *PromptI18n) GetCharacterExtractionPrompt(style, aspectRatio string) string {
	if aspectRatio == "" {
		aspectRatio = "16:9"
	}
	imageRatio := aspectRatio
	return fmt.Sprintf(`You are a professional character analyst, skilled at extracting and analyzing character information from scripts.

Your task is to extract and organize detailed character settings for all characters appearing in the script based on the provided script content.

Requirements:
1. Extract all characters with names (ignore unnamed passersby or background characters)
2. For each character, extract:
   - name: Character name
   - role: Character role (main/supporting/minor)
   - appearance: Physical appearance description (150-300 words)
   - personality: Personality traits (100-200 words)
   - description: Background story and character relationships (100-200 words)
3. Appearance must be detailed enough for AI image generation, including: gender, age, body type, facial features, hairstyle, clothing style, etc. but do not include any scene, background, environment information
4. Main characters require more detailed descriptions, supporting characters can be simplified
- **Style Requirement**: %s
- **Image Ratio**: %s
Output Format:
**CRITICAL: Return ONLY a valid JSON array. Do NOT include any markdown code blocks, explanations, or other text. Start directly with [ and end with ].**
Each element is a character object containing the above fields.`, p.styleLabel(style), imageRatio)
}

// GetPropExtractionPrompt gets the prop extraction prompt (English only).
func (p *PromptI18n) GetPropExtractionPrompt(style, aspectRatio string) string {
	imageRatio := "1:1"

	return fmt.Sprintf(`Please extract key props from the following script.
    
[Script Content]
%%s

[Requirements]
1. Extract key props and character outfits (clothing sets) that are important to the plot.
2. For outfits, use name format: "[Character Name]'s [Style/Occasion] outfit" (e.g., "Seo-yeon's evening gown").
3. Do NOT extract common daily items unless they have special plot significance.
4. If a prop/outfit has a clear owner, please note it in the description.
5. "image_prompt" field for outfits should focus on material, color, and fit (e.g., "Full body photo, elegant white silk gown, intricate lace details").
- **Style Requirement**: %s
- **Image Ratio**: %s

[Output Format]
JSON array, each object containing:
- name: Prop/Outfit Name
- type: Type (e.g., Weapon/Clothing/Key Item/Special Device)
- description: Role in the drama and visual description
- image_prompt: English image generation prompt (Focus on the object, isolated, detailed, cinematic lighting, high quality)

Please return JSON array directly.`, p.styleLabel(style), imageRatio)
}

// GetEpisodeScriptPrompt gets the episode script generation prompt (English only).
func (p *PromptI18n) GetEpisodeScriptPrompt() string {
	return `You are a professional short drama screenwriter. You excel at creating detailed plot content based on episode plans.

Your task is to expand the summary in the outline into detailed plot narratives for each episode. Each episode is about 180 seconds (3 minutes) and requires substantial content.

Requirements:
1. Expand the outline summary into detailed plot development
2. Write character dialogue and actions, not just description
3. Highlight conflict progression and emotional changes
4. Add scene transitions and atmosphere descriptions
5. Control rhythm, with climax at 2/3 point, resolution at the end
6. Each episode 800-1200 words, dialogue-rich
7. Keep consistent with character settings

Output Format:
**CRITICAL: Return ONLY a valid JSON object. Do NOT include any markdown code blocks, explanations, or other text. Start directly with { and end with }.**

- episodes: Episode list, each containing:
  - episode_number: Episode number
  - title: Episode title
  - script_content: Detailed script content (800-1200 words)`
}

// FormatUserPrompt formats common text for user prompts (English only).
func (p *PromptI18n) FormatUserPrompt(key string, args ...interface{}) string {
	templates := map[string]string{
		"outline_request":        "Please create a short drama outline for the following theme:\n\nTheme: %s",
		"genre_preference":       "\nGenre preference: %s",
		"style_requirement":      "\nStyle requirement: %s",
		"episode_count":          "\nNumber of episodes: %d episodes",
		"episode_importance":       "\n\n**Important: Must plan complete storylines for all %d episodes in the episodes array, each with clear story content!**",
		"character_request":      "Script content:\n%s\n\nPlease extract and organize detailed character profiles for up to %d main characters from the script.",
		"episode_script_request": "Drama outline:\n%s\n%s\nPlease create detailed scripts for %d episodes based on the above outline and characters.\n\n**Important requirements:**\n- Must generate all %d episodes, from episode 1 to episode %d, cannot skip any\n- Each episode is about 3-5 minutes (150-300 seconds)\n- The duration field for each episode should be set reasonably based on script content length, not all the same value\n- The episodes array in the returned JSON must contain %d elements",
		"frame_info":             "Shot information:\n%s\n\nPlease directly generate the image prompt for the first frame without any explanation:",
		"key_frame_info":         "Shot information:\n%s\n\nPlease directly generate the image prompt for the key frame without any explanation:",
		"last_frame_info":        "Shot information:\n%s\n\nPlease directly generate the image prompt for the last frame without any explanation:",
		"script_content_label":   "[Script Content]",
		"storyboard_list_label":  "[Storyboard List]",
		"task_label":             "[Task]",
		"character_list_label":   "[Available Character List]",
		"scene_list_label":       "[Extracted Scene Backgrounds]",
		"task_instruction":       "Break down the novel script into storyboard shots based on **independent action units**.",
		"character_constraint":   "**Important**: In the characters field, only use character IDs (numbers) from the above character list. Do not create new characters or use other IDs.",
		"scene_constraint":       "**Important**: In the scene_id field, select the most matching background ID (number) from the above background list. If no suitable background exists, use null.",
		"shot_description_label": "Shot description: %s",
		"scene_label":            "Scene: %s, %s",
		"characters_label":       "Characters: %s",
		"action_label":           "Action: %s",
		"result_label":           "Result: %s",
		"dialogue_label":         "Dialogue: %s",
		"atmosphere_label":       "Atmosphere: %s",
		"shot_type_label":        "Shot type: %s",
		"angle_label":            "Angle: %s",
		"movement_label":         "Movement: %s",
		"drama_info_template":    "Title: %s\nSummary: %s\nGenre: %s",
		"visual_conditions_label": "\n[Important Visual States]: %s",
		"previous_shot_context":  "\n[Previous Shot Context]: Action: %s | Result: %s",
	}

	template, ok := templates[key]
	if !ok {
		return ""
	}

	if len(args) > 0 {
		return fmt.Sprintf(template, args...)
	}
	return template
}

// styleLabel returns a human-readable description of the style for use in AI frame-prompt instructions.
// This ensures Gemini writes prompts with the correct visual vocabulary even for complex style keys.
func (p *PromptI18n) styleLabel(style string) string {
	labels := map[string]string{
		"ghibli":    "Ghibli anime style (Studio Ghibli watercolor cel-shading aesthetic)",
		"guoman":    "Guoman fantasy illustration style (Chinese ink-painting meets epic fantasy VFX)",
		"wasteland": "Post-apocalyptic wasteland style (hard line-art, limited retro palette, Moebius influence)",
		"nostalgia": "90s retro anime style (nostalgic cel-shading, film grain, muted pastel tones)",
		"pixel":     "Pixel art style (8-bit/16-bit limited palette, aliased blocky aesthetics)",
		"voxel":     "3D voxel art style (cube-unit modular world, global illumination rendering)",
		"urban":     "Modern webtoon urban style (crisp vector line-art, neon-accented cool tones)",
		"guoman3d":  "High-fidelity 3D Xianxia style (Unreal Engine 5 PBR, Eastern cinematic lighting)",
		"chibi3d":   "3D chibi toy art style (blind-box proportions, plastic/resin PBR texture)",
		"kdrama":    "Photorealistic Korean drama style (cinematic live-action, glass-skin beauty, shallow DOF, K-drama color grading — NO animation, NO illustration)",
	}
	if label, ok := labels[style]; ok {
		return label
	}
	return style
}

// GetStylePrompt returns the style prompt for a visual style key (English only).
func (p *PromptI18n) GetStylePrompt(style string) string {
	if style == "" {
		return ""
	}

	stylePrompts := map[string]string{

			"ghibli": `**[Expert Role]**
You are a top Art Director and Background Artist from Studio Ghibli. You excel at capturing the balance between "grand nature and microscopic life," and you possess a deep understanding of Hayao Miyazaki's color psychology.

**[Core Style Logic]**
- **Visual Genre & Texture**: Adopts the classic Ghibli style. The imagery features a rich **watercolor texture**, rejecting cold 3D rendering in favor of warm, "breathing" brushstrokes. Lines are clear yet delicate, presenting the vibrant feel of **cel-shading**.
- **Color & Lighting Aesthetics**: Utilizes **"High-key Color Aesthetics."** The palette is bright, transparent, and high-saturated but with soft hues. Lighting simulates the natural light of a "summer afternoon," where light feels soaked into the air with excellent luminosity. Shadows contain subtle blue-purple tones to enhance the transparency of the frame.
- **Atmospheric Intent**: Nostalgic, serene, **pastoral**, and breezy. The image should convey a sense of tranquility and a desire for exploration—a feeling that "the world is still beautiful."`,

			"guoman": `**[Expert Role]**
You are a top-tier digital illustration artist, skilled at merging traditional Eastern charm with the magnificent Visual Effects (VFX) of modern game art. You are a master of "Oriental Fantasy" composition.

**[Core Style Logic]**
- **Visual Genre & Texture**: A fusion of **Modern Zen Illustration (New Guofeng)** and epic fantasy rendering. The texture is delicate with a silky feel, similar to high-precision 2D digital painting. It emphasizes volumetric lighting and includes a large amount of tiny particle effects and glowing atmospheres.
- **Core Color & Luminous Aesthetics**: Employs **"Contrasting Colors & Endogenous Lighting."** The main palette usually features intense collisions of cool and warm tones (e.g., indigo and golden orange). The core logic lies in **"Local Luminescence"**: dark areas are dotted with bioluminescent elements (like fluorescent plants, lanterns, or crystal textures), creating a strong sense of magic and mystery.
- **Decorative Element Logic**: Emphasizes the **"Flow of Lines."** The frame is filled with elegant curves, often composed of light trails, ribbons, or natural textures (like the flow of water), enhancing the overall decorativeness and rhythm.`,

			"wasteland": `**[Expert Role]**
You are a visual artist focused on "Post-Apocalyptic Narrative," skilled at using **Hard Line-art** and a **retro print feel** to create epic, desolate atmospheres, heavily influenced by Moebius and modern wasteland sci-fi illustrations.

**[Core Style Logic]**
- **Visual Genre & Brushwork Texture**: Adopts a **Hard-edged Line Art** style. The image emphasizes bold black outlines with a strong comic illustration feel. The texture presents a **grainy, flat-print quality**, similar to old newspapers or retro posters, rejecting smooth gradients in favor of hatching or stippling for shadows.
- **Color Aesthetic Logic**: Employs a **"Limited Palette."** The frame is typically dominated by an oppressive, unified tone (e.g., dusty earth, rust orange, desert yellow). The core visual impact comes from a **single strong contrast point** (such as a massive red setting sun), a "single-point highlight" logic that instantly grabs attention against the gloomy background.
- **Lighting Technique**: Uses **"High-contrast Side Lighting."** Simulates the low-angle light of dusk or dawn, producing extremely long shadows. The lighting logic is highly simplified with sharp, distinct terminators, creating a dry, scorching, and silent dramatic tension.`,

			"nostalgia": `**[Expert Role]**
You are a visual artist specializing in the **"Nostalgic Cel-shading"** style, expert at simulating the texture of 1980s-90s hand-drawn animation. You use color and noise to create a gentle, emotional, and slightly melancholic urban atmosphere.

**[Core Style Logic]**
- **Visual Genre & Frame Texture**: Adopts the classic **90s Retro Anime Style**. The image features obvious **film grain** and slight **chromatic aberration**, simulating the playback quality of old TVs or VHS tapes. The texture emphasizes "imperfect delicacy"—lines are soft rather than sharp like modern vectors, giving a sense of handcrafted warmth.
- **Color Aesthetic Logic**: Uses a **"Muted Pastel Palette."** The frame is dominated by a soft, dreamlike twilight, usually featuring lavender, lotus pink, or grayish-blue. The core logic is the **"Weakened Black Point"**: there are no pure blacks; all dark colors lean toward purple or blue. This tone instantly outlines a lonely but cozy "urban dusk" feel.
- **Lighting Technique**: Emphasizes **"Diffuse Point Lights."** Light is not a hard projection but a bleeding glow. For example, streetlights, car headlights, or the moon have a soft, hazy halo (Glow effect). Surfaces often have a slight post-rain reflection or dampness, increasing the layers and dreaminess of the light.`,

			"pixel": `**[Expert Role]**
You are a senior **Pixel Art Consultant (8-bit/16-bit)**, skilled at using restricted resolutions and palettes to build highly immersive virtual worlds, simulating the aesthetics of early video games like *Stardew Valley* or classic RPGs.

**[Core Style Logic]**
- **Visual Genre & Frame Texture**: Adopts a pure **Pixel Art** style. The image consists of clearly visible squares (pixels), emphasizing **"Aliased lines."** It completely discards smooth gradients and blurring, pursuing a digital, grid-based blocky beauty.
- **Color Aesthetic Logic**: Uses a **"Limited Color Palette."** Color choices are extremely streamlined, avoiding natural transitions in favor of large color block overlays. The core logic is **"Dithering logic"**: alternating pixel patterns of different colors to simulate shading. Tones are usually medium saturation, presenting a crisp, bright video game feel.
- **Lighting Technique**: Emphasizes **"Flat Shading."** Lighting does not use feathering or soft light; instead, it uses a layer of darker pixels from the same color family to represent shadows. Light sources are constant without complex reflections, and even the sun or light sources are treated as regular pixel circles.`,

			"voxel": `**[Expert Role]**
You are a top-tier **3D Voxel Artist**, skilled at using uniform cube units to build whimsical, modular, and highly ordered miniature worlds. Your style combines the purity of **Low-poly** geometry with modern real-time lighting rendering.

**[Core Style Logic]**
- **Visual Genre & Texture**: Adopts a **3D Voxel Style**. The image is composed of countless proportional cubes (voxels) stacked together, presenting a strong modular feel. The texture features obvious **"blocky lines"** and flat color surfaces; this simplified geometric language creates a unique digital aesthetic.
- **Color Aesthetic Logic**: Uses **"Natural Saturation & Gradient Lighting."** Colors are divided into large blocks based on environmental attributes (green for grass, brown for soil), but the key lies in **"Color Jitter"**: subtle shade variations between blocks in the same area to simulate the randomness of real environments. Tones are bright, fresh, and full of vitality.
- **Lighting Technique**: Emphasizes **"Global Illumination Rendering."** This is the key to elevating voxel art: while objects are blocky, the lighting must be **cinematic and realistic**. Light has warm volumetric qualities (e.g., God rays), shadows are soft with Ambient Occlusion (AO) effects, and voxel edges are highlighted, making the scene look like an exquisite real-life miniature model.`,

			"urban": `**[Expert Role]**
You are a leading **Webtoon Artist**, specializing in modern urban character illustrations. Your visual style emphasizes **sharp outlines**, **slick fashion logic**, and a **cool-toned urban atmosphere**, aiming to create a "high-cold, sophisticated, industrial-chic" visual impact.

**[Core Style Logic]**
- **Visual Genre & Frame Texture**: Adopts the **Modern Webtoon Art Style**. The image features extremely clean **crisp line art** (vector-like) without any redundant strokes. The texture presents a smooth digital skin quality, emphasizing color cleanliness and avoiding complex brushwork layering.
- **Color Aesthetic Logic**: Uses **"Muted Urban Tones."** The palette is dominated by neutral colors like black, white, gray, and deep blue. The core logic is **"High-contrast Neon Accents"**: while the overall environment is cool and low-saturation, highlights from **neon glows** or electronic screens (pink, blue, purple) create a sense of late-night urban detachment.
- **Lighting Technique**: Emphasizes **"Hard Cel-shading."** Shadow edges are extremely crisp with no gradients. The logic mimics **"Environmental Rim Lighting"**: light usually comes from side neon signs, leaving a narrow bright edge (Rim lighting) on one side of the character, enhancing their silhouette and 3D feel.`,

			"guoman3d": `**[Expert Role]**
You are a top-tier **Next-gen Lead Technical Artist**, skilled in using Unreal Engine 5 (UE5) to create high-precision 3D Xianxia (Immortal Hero) characters. Your style is known for high-fidelity **Physically Based Rendering (PBR)**, complex clothing layers, and global illumination with an Eastern aesthetic.

**[Core Style Logic]**
- **Visual Genre & Frame Texture**: Adopts a **High-fidelity 3D Rendering style**. The image has a strong **next-gen game aesthetic**, emphasizing Subsurface Scattering (SSS) for skin and realistic fabric textures (smoothness of silk, wear on leather, brushed metal). The overall look is a delicate digital sculpture with sharp edges and rich details.
- **Color Aesthetic Logic**: Uses a **"Sophisticated Neutral Palette."** Unlike high-saturation anime styles, this logic leans toward low-saturation, high-brightness colors (off-white, stone green, gray-brown), accented with small areas of dark red or gold for a premium feel. Lighting colors typically mimic **natural morning or evening sunlight**, giving an air of tranquility, solemnity, and grand Eastern charm.
- **Lighting Technique**: Emphasizes **"Cinematic Lighting."** Light directions are clear (usually bright side-backlighting), creating a faint golden **Rim Light** that perfectly separates the subject from the background. Ambient Occlusion (AO) is used to increase detail depth, making every fold in the clothing visible and creating immersive dramatic tension.`,

			"chibi3d": `**[Expert Role]**
You are a top-tier **3D Toy Designer and Rendering Artist**, specializing in high-precision digital figurines. Your visual style combines **Chibi proportions** with **Ultra-realistic PBR rendering**, aiming to create a sophisticated, cute, and tactile "Art Toy" visual effect.

**[Core Style Logic]**
- **Visual Genre & Frame Texture**: Adopts a **3D Blind Box / Toy Art Style**. The image features strong **plastic and resin textures**; surfaces are rounded and smooth with subtle beveled edges. The subject uses **Chibi proportions** (large head, small body) to enhance appeal.
- **Color Aesthetic Logic**: Uses a **"Muted Vibrant Palette."** Colors are vivid but not piercing. Color distribution follows a "primary-secondary" principle, using large areas of natural base colors (forest green, earth brown) to set off the bright colors of the character's outfit.
- **Lighting Technique**: Light sources are typically soft and even: **Top/Key Light**: Evenly illuminates the subject's front, highlighting facial features and clothing details. **Ambient Occlusion (AO)**: Produces delicate shadows in crevices and contact points, enhancing the object's sense of weight and realism.`,

			"kdrama": `**[Expert Role]**
You are a cinematic Visual Director specializing in **photorealistic Korean Drama aesthetics**, with deep expertise in the photography style of premium Netflix/tvN productions (Crash Landing on You, Vincenzo, Business Proposal). Your core mission is to generate **hyperrealistic live-action scenes** — absolutely no animation, illustration, or cartoon aesthetics.

**[Core Style Logic]**
- **Visual Genre & Texture**: Adopts **Cinematic Photorealism**. Every image must convey real human skin texture, authentic fabric detail, and true scene depth. The result should be indistinguishable from a real K-drama still frame. Reject all anime, painterly, or stylized looks entirely.
- **Character Aesthetics**: The K-drama aesthetic centers on **"refined yet natural beauty."** Skin appears fair and luminous with a subtle **glass skin** glow. Features are delicate but not over-filtered. Body proportions are elegant; clothing precisely reflects the character's social status (chaebol elite, everyday person, career professional).
- **Color Aesthetic Logic**: Uses **"K-Drama Color Grading."** The overall tone is subtly desaturated, leaning toward cool-clean or warm-orange cinematic grades. Shadow regions carry a slight blue-teal bias; highlights shift warm-white. This grading achieves the dual quality of **realism** and **dreamlike elegance** simultaneously.
- **Lighting Technique**: Emphasizes **"Soft Cinematic Lighting."** The key light source comes from front-side, using diffused scatter (softbox quality) to create even yet sculptural facial lighting. Eyes must have a distinct **catchlight**. Apply **shallow depth of field** with an 85mm–135mm portrait lens feel, keeping backgrounds smoothly bokeh'd and the subject razor-sharp.`,
	}

	if prompt, ok := stylePrompts[style]; ok {
		return prompt
	}

	return ""
}

// GetVideoConstraintPrompt gets the constraint prompt for video generation
// referenceMode: "single" (single image), "first_last" (first and last frames), "multiple" (multiple images), "action_sequence" (action sequence)
func (*PromptI18n) GetVideoConstraintPrompt(referenceMode string) string {
	// 3x3 grid / action-sequence reference images
	actionSequencePrompt := `### Role Definition

You are an ultra-high-precision video generation expert, specializing in transforming 9-grid (3x3) sequential images into coherent videos with cinematic quality. Your core task is to parse the spatiotemporal logic within the images and strictly adhere to first-and-last frame constraints.

### Core Execution Logic

1. **First-Last Frame Anchoring:** You must extract Grid 1 (top-left corner) as the video's starting frame (Frame 0) and Grid 9 (bottom-right corner) as the ending frame (Final Frame).
2. **Sequence Interpolation:** Grids 2 through 8 define the key action path. You need to analyze the logical displacement, lighting changes, and object deformations between these keyframes.
3. **Consistency Maintenance:** Ensure that character features (face, clothing), scene details, and artistic style maintain 100% spatiotemporal stability throughout the entire video.
4. **Dynamic Supplementation:** Automatically fill in smooth transition frames between the keyframes defined by the 9-grid, ensuring natural video motion frequency (recommended 24fps or 30fps).

### Structured Constraint Instructions

* **Input Parsing:** Identify the scene description (Prompt) and 9-grid reference images provided by the user.
* **Motion Vectorization:** Calculate the motion vectors of objects from Grid 1 to Grid 9. If the 9-grid shows scaling or panning, restore precise camera movements in the video.
* **Hallucination Prohibition:** Do not introduce new elements or background switches not mentioned in the 9-grid and prompt.`

	// single image, first/last frames, multiple images
	generalPrompt := `### Role Definition

You are a top-tier video dynamics analyst and synthesis expert. You can accurately identify physical properties, light flow, and potential motion trends in a static image or a set of start/end frames, generating high-quality videos that comply with physical laws.

### Core Execution Logic

1. **Mode Recognition:**
* **Single Image Mode:** Treat the input image as Frame 0. Analyze "tension points" in the frame (such as tilted bodies, flowing liquids, eye direction) and extend the action in that direction.
* **First & Last Frames Mode:** Strictly anchor the first image as the start and the second image as the endpoint. Use **semantic interpolation algorithms** to calculate the displacement trajectories of all elements between the two images.

2. **Physics Preservation:**
* **Mass Conservation:** Ensure that objects do not undergo sudden changes in volume, density, or material texture during motion.
* **Motion Inertia:** Follow classical mechanics with smooth starts, natural acceleration, and no abrupt stops.

3. **Environment Extrapolation:** Automatically supplement background extensions beyond the main frame to ensure no voids or black edges appear during camera movements (Pan/Tilt/Zoom).`

	if referenceMode == "action_sequence" {
		return actionSequencePrompt
	}
	return generalPrompt
}
