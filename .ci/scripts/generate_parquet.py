#!/usr/bin/env python3
"""
Generate test_data/articles.parquet — 10,000 article records in article format.
Parquet schema maps to cmdChroma import flags:
  --field-content body    (document content)
  --field-id id           (document ID)
  --field-metadata title,category,source
"""

import pandas as pd
import random

random.seed(42)

CATEGORIES = {
    "technology": {
        "topics": [
            ("How Vector Databases Work",
             "A vector database stores data as high-dimensional vectors, enabling semantic similarity search. "
             "Unlike traditional databases that query exact matches, vector databases find items closest to a query vector using distance metrics like cosine similarity or L2 distance. "
             "This enables applications such as semantic search, recommendation systems, and retrieval-augmented generation. "
             "Popular implementations include ChromaDB, Pinecone, Weaviate, and Qdrant. "
             "Vector databases are optimized forANN (Approximate Nearest Neighbor) lookups, making them scalable to billions of embeddings."),
            ("Understanding RAG Pipelines",
             "Retrieval-Augmented Generation combines a retrieval system with a large language model to produce context-aware answers. "
             "The pipeline ingests documents, generates vector embeddings, and stores them in a vector database. "
             "At query time, the user's question is embedded and semantically searched against stored vectors. "
             "The retrieved documents are injected into the LLM prompt as context, improving factual accuracy and reducing hallucinations. "
             "RAG pipelines are essential for enterprise AI applications that need answers grounded in specific knowledge bases."),
            ("WebAssembly in Production",
             "WebAssembly (Wasm) is a binary instruction format that enables near-native performance in browsers and servers. "
             "Initially designed for the web, Wasm has expanded to serverless functions, edge computing, and embedded systems. "
             "Key benefits include portability, security sandboxing, and language interoperability. "
             "The WebAssembly System Interface (WASI) standardizes system calls, enabling Wasm modules to interact with files, networks, and clocks. "
             "Runtime environments like Wasmtime, WasmEdge, and WAMR support running Wasm outside browsers."),
            ("Container Orchestration with Kubernetes",
             "Kubernetes automates the deployment, scaling, and management of containerized applications across clusters of machines. "
             "It abstracts away infrastructure complexity, allowing developers to define desired states for their applications. "
             "Key concepts include Pods (smallest deployable units), Services (network abstractions), Deployments (declarative updates), "
             "and ConfigMaps/Secrets (configuration management). "
             "Kubernetes supports horizontal pod autoscaling, rolling updates, and self-healing through automatic restart of failed containers. "
             "The Kubernetes API is extensible through Custom Resource Definitions and operators."),
            ("API Design Principles for Modern Services",
             "Well-designed APIs are consistent, intuitive, and evolve gracefully over time. "
             "REST remains the dominant paradigm for web APIs, using HTTP methods and resource-oriented URLs. "
             "GraphQL offers flexibility by letting clients specify exactly which fields they need. "
             "gRPC, built on HTTP/2 and Protocol Buffers, excels for internal microservice communication with its binary protocol. "
             "Regardless of protocol, good API design prioritizes clear naming, comprehensive error responses, "
             "versioning strategy, and thorough documentation. "
             "API gateways handle cross-cutting concerns like authentication, rate limiting, and request routing."),
        ],
        "sources": ["devto", "stackoverflow", "github", "medium", "hackernoon"],
    },
    "science": {
        "topics": [
            ("The Physics of Black Holes",
             "Black holes are regions of spacetime where gravity is so strong that nothing, not even light, can escape once past the event horizon. "
             "They form when massive stars (at least 20 solar masses) exhaust their nuclear fuel and collapse under their own gravity. "
             "The Schwarzschild radius defines the event horizon: Rs = 2GM/c^2, where G is the gravitational constant. "
             "Black holes spin due to the conservation of angular momentum from their progenitor stars. "
             "The fastest known black hole, MAXI J1348-630, spins at about 86% of the theoretical maximum. "
             "Hawking radiation is a theoretical quantum effect where black holes slowly emit thermal radiation over immense timescales."),
            ("CRISPR Gene Editing Explained",
             "CRISPR-Cas9 is a molecular toolkit derived from bacterial immune systems that cuts DNA at specific locations. "
             "The Cas9 protein acts as molecular scissors, guided by a complementary RNA sequence to target precise genomic locations. "
             "This enables researchers to disable genes, insert new sequences, or correct mutations associated with genetic diseases. "
             "Clinical trials are underway for sickle cell disease, blindness, cancer immunotherapy, and HIV. "
             "Ethical concerns include germline editing (heritable changes), off-target effects, and equitable access to treatments. "
             "The 2020 Nobel Prize in Chemistry was awarded to Jennifer Doudna and Emmanuelle Charpentier for CRISPR discovery."),
            ("The Scale of the Observable Universe",
             "The observable universe spans approximately 93 billion light-years in diameter, containing roughly 2 trillion galaxies. "
             "This enormous scale arises because the universe has been expanding for 13.8 billion years, carrying distant objects far beyond "
             "what light could have traveled in that time. The cosmic microwave background, at 380,000 years old, is the oldest light we can see. "
             "Most of the universe's energy density is dark energy (68%), driving accelerated expansion, while dark matter (27%) "
             "provides the gravitational scaffolding for galaxies. Ordinary matter — everything we can see — makes up only 5%. "
             "The Milky Way contains 100-400 billion stars, and the nearest star system, Alpha Centauri, is 4.37 light-years away."),
            ("Climate Modeling and Prediction",
             "Climate models divide Earth's surface and atmosphere into a 3D grid, simulating fluid dynamics, radiative transfer, and biogeochemistry. "
             "General Circulation Models (GCMs) solve partial differential equations for atmospheric motion at resolutions from 50 to 250 km. "
             "Coupled Model Intercomparison Project (CMIP) runs coordinate climate simulations across dozens of institutions worldwide. "
             "Key feedback mechanisms include ice-albedo feedback, water vapor amplification, and permafrost carbon release. "
             "Representative Concentration Pathways (RCPs) and Shared Socioeconomic Pathways (SSPs) define future emission scenarios. "
             "Modern models now incorporate machine learning components for surrogate modeling and downscaling projections."),
            ("Quantum Entanglement in Experiments",
             "Quantum entanglement is a phenomenon where two particles share a quantum state such that measuring one instantly affects the other, "
             "regardless of distance. Einstein famously called this 'spooky action at a distance.' "
             "Bell test experiments, beginning with John Bell's 1964 theorem, have repeatedly violated Bell inequalities, ruling out local hidden variable theories. "
             "The 2022 Nobel Prize in Physics honored Alain Aspect, John Clauser, and Anton Zeilinger for experiments with entangled photons. "
             "Entanglement enables quantum cryptography, quantum teleportation, and quantum computing. "
             "Decoherence — interaction with the environment — is the primary challenge in maintaining entanglement in quantum systems. "
             "Recent experiments have demonstrated entanglement over fiber distances exceeding 1,200 km via satellite relay."),
        ],
        "sources": ["nature", "sciencedaily", "wikipedia", "space.com", "popsci"],
    },
    "health": {
        "topics": [
            ("Understanding Cardiovascular Health",
             "The cardiovascular system pumps approximately 5 liters of blood per minute at rest, increasing to 20-25 liters during exercise. "
             "The heart's sinoatrial node sets the natural pacemaker rhythm at 60-100 beats per minute. "
             "Cardiovascular disease (CVD) remains the leading cause of death globally, accounting for 17.9 million deaths annually. "
             "Modifiable risk factors include hypertension, smoking, diabetes, obesity, physical inactivity, and diet. "
             "The DASH diet and Mediterranean diet have strong evidence for reducing blood pressure and CVD risk. "
             "Aerobic exercise strengthens the myocardium, improves endothelial function, and increases HDL cholesterol."),
            ("Sleep Architecture and Why It Matters",
             "Sleep progresses through four stages: three NREM stages (N1, N2, N3) and REM sleep, cycling every 90-110 minutes. "
             "N1 is light sleep (5%), N2 is light sleep with sleep spindles and K-complexes (50%), and N3 (slow-wave sleep) is deep restorative sleep (20%). "
             "REM sleep, constituting 25% of total sleep, is associated with memory consolidation, emotional processing, and dreaming. "
             "Adults need 7-9 hours of sleep per night. Chronic sleep deprivation impairs glucose metabolism, immune function, and cognitive performance. "
             "Sleep apnea, affecting 936 million people worldwide, disrupts breathing during sleep and significantly increases cardiovascular risk. "
             "Melatonin secretion is controlled by the suprachiasmatic nucleus in response to darkness, regulating circadian rhythms."),
            ("The Gut Microbiome's Role in Health",
             "The human gut harbors approximately 38 trillion microbial cells, outnumbering human cells by a ratio of roughly 1.3:1. "
             "Over 1,000 bacterial species have been identified in the human gut, with Prevotella, Bacteroides, and Faecalibacterium among the most common genera. "
             "The microbiome influences digestion, vitamin synthesis (K and B vitamins), immune system development, and neurotransmitter production "
             "(95% of the body's serotonin is produced in the gut). "
             "Dysbiosis — an imbalanced microbiome — has been linked to inflammatory bowel disease, obesity, type 2 diabetes, depression, and autism. "
             "Diet strongly shapes microbiome composition. High-fiber, fermented-food diets increase microbial diversity; "
             "high-sugar, high-fat diets reduce it. Probiotics and prebiotics are dietary interventions targeting microbiome health."),
            ("Mental Health: Anxiety and Depression",
             "Anxiety disorders are the most common mental health conditions, affecting 284 million people globally. "
             "Generalized anxiety disorder (GAD) involves persistent, excessive worry about everyday situations. "
             "Depression (major depressive disorder) affects 264 million people and is characterized by persistent sadness, anhedonia, and cognitive impairment. "
             "Both conditions involve dysregulation of neurotransmitters: serotonin, norepinephrine, and dopamine. "
             "First-line treatments include Cognitive Behavioral Therapy (CBT) and SSRIs (selective serotonin reuptake inhibitors). "
             "Lifestyle interventions — regular exercise, sleep hygiene, and social connection — have effect sizes comparable to medication for mild-to-moderate depression. "
             "ketamine and psilocybin are emerging as rapid-acting treatments for treatment-resistant depression."),
            ("Nutrition Science: What We Actually Know",
             "Nutrition research faces unique methodological challenges, including recall bias in food frequency questionnaires, "
             "difficulty blinding participants, and confounds from linked lifestyle factors. "
             "Randomized controlled trials are the gold standard but are often impractical for long-term dietary outcomes. "
             "The PREDIMED trial (7,447 participants) demonstrated that Mediterranean diet reduced cardiovascular events by 30%. "
             "Evidence strongly supports: limiting saturated fats (replacing with unsaturated fats), reducing added sugars, "
             "eating fiber-rich whole foods, and maintaining adequate omega-3 intake. "
             "Controversies persist around optimal protein intake, vitamin D supplementation, and the health effects of red meat. "
             "Nutrition epidemiology is increasingly using biomarkers, mobile apps, and continuous glucose monitors to improve measurement accuracy."),
        ],
        "sources": ["who", "nih", "mayoclinic", "healthline", "webmd"],
    },
    "business": {
        "topics": [
            ("Agile Transformation at Scale",
             "Agile methodologies, originating from the 2001 Manifesto for Agile Software Development, emphasize iterative delivery, "
             "customer collaboration, and responding to change over rigid planning. "
             "Scaling Agile beyond single teams requires frameworks like SAFe (Scaled Agile Framework), LeSS (Large-Scale Scrum), or Spotify's model. "
             "Common causes of Agile transformation failure include: lack of executive sponsorship, treating Agile as a process rather than a mindset shift, "
             "insufficient training, and siloed team structures that contradict Agile principles. "
             "Spotify's model uses Squads (cross-functional teams), Tribes (collections of squads), Chapters (functional leads), and Guilds (communities of interest). "
             "Measuring Agile success involves tracking velocity trends, sprint burndown, teamappiness, and delivery predictability."),
            ("Data-Driven Decision Making",
             "Data-driven organizations use evidence rather than intuition alone to guide strategy, prioritize features, and allocate resources. "
             "Key practices include: defining clear metrics before building, automating data collection, using A/B testing to validate assumptions, "
             "and establishing data quality processes. "
             "The 'flywheel' concept — where data generates insights, insights improve products, more products generate more data — creates compounding competitive advantage. "
             "Common pitfalls include: confusing correlation with causation, p-hacking, survivorship bias in performance metrics, "
             "and 'vanity metrics' that look impressive but don't drive decisions. "
             "Data governance frameworks address data ownership, access control, lineage tracking, and compliance (GDPR, CCPA)."),
            ("Supply Chain Resilience After COVID",
             "The COVID-19 pandemic exposed critical vulnerabilities in globally optimized supply chains: single-source dependencies, "
             "just-in-time inventory that left no buffer, and geographic concentration in low-cost regions prone to disruption. "
             "Resilience strategies include: supply chain diversification, nearshoring/friendshoring, safety stock increases, "
             "dual-sourcing critical components, and digital twin simulation for disruption ScenarioPlanning. "
             "The 2021 semiconductor shortage cost the automotive industry an estimated $210 billion in lost revenue. "
             "Supply chain visibility platforms (project44, FourKites, Resilinc) use IoT and AI to track shipments in real time. "
             "Environmental, Social, and Governance (ESG) reporting requirements are driving supply chain transparency initiatives."),
            ("Product-Led Growth Strategies",
             "Product-led growth (PLG) places the product at the center of acquisition, conversion, and retention — instead of sales or marketing. "
             "Key PLG mechanisms include: free tiers or freemium models, in-product collaboration, virality loops, and seamless onboarding. "
             "Successful PLG companies include Slack, Figma, Notion, Dropbox, and Atlassian (with Jira). "
             "PLG requires exceptional product experience (PX) — onboarding must be frictionless, time-to-value must be under 5 minutes, "
             "and the product must sell itself through clear value demonstration. "
             "The 'land and expand' motion starts with individual or team adoption, then expands to department and enterprise level through product-led sales. "
             "Metrics: time-to-value (TTV), viral coefficient, net revenue retention (NRR), and feature adoption rates."),
            ("Corporate Innovation and Corporate Venturing",
             "Large corporations face the innovator's dilemma: their existing business model, customer base, and culture can prevent them from "
             "capitalizing on disruptive innovations. "
             "Corporate venturing addresses this through internal innovation labs, accelerator partnerships, venture capital investments, "
             "acquisitions of startups, and intrapreneurship programs. "
             "Examples: Alphabet's X (moonshot factory), Apple's acquisition-driven strategy (Shazam, Siri, Touch ID), "
             "Salesforce's Salesforce Ventures, and Toyota's Woven Planet for autonomous vehicles. "
             "Success factors for corporate innovation include: autonomous governance (shielded from bureaucracy), "
             "clear Stage-Gate evaluation processes, and willingness to kill projects that aren't working. "
             "Failure modes: corporate venture arms that are purely financial (lose strategic alignment) or overly controlled (stifled innovation)."),
        ],
        "sources": ["hbr", "mckinsey", "forbes", "fastcompany", "businessinsider"],
    },
    "history": {
        "topics": [
            ("The Fall of the Roman Empire",
             "The Western Roman Empire fell in 476 CE when the Germanic chieftain Odoacer deposed the last emperor, Romulus Augustulus. "
             "This was not a single event but the culmination of centuries of structural problems: political instability "
             "(26 emperors in the 3rd century alone), economic strain from military spending, currency debasement causing inflation, "
             "population decline from plagues, and pressure from migrating barbarian groups. "
             "The Eastern Roman (Byzantine) Empire survived until 1453. "
             "Historian Edward Gibbon's 'The History of the Decline and Fall of the Roman Empire' (1776) attributed the fall primarily to Christianity, "
             "a view largely rejected by modern historians who cite multi-causal systemic collapse. "
             "Rome's legacy endures in law, language, architecture, governance, and the concept of western civilization itself."),
            ("The Silk Road's Influence on Civilization",
             "The Silk Road was a network of trade routes connecting East Asia (China, India) to the Mediterranean, active from roughly 130 BCE to 1453 CE. "
             "More than silk traveled these routes: technologies (paper, printing, gunpowder, compass), diseases (bubonic plague), religions ( Buddhism, Islam), "
             "and artistic styles spread between civilizations. "
             "The peak of the Silk Road coincided with the Pax Mongolica (13th-14th centuries), when Mongol expansion created safe corridors across Eurasia. "
             "The route declined after the Ottoman conquest of Constantinople disrupted Mediterranean trade and when European sea routes "
             "(Vasco da Gama, 1498) provided cheaper oceanic alternatives. "
             "Archaeological sites along the route include Samarkand, Dunhuang's Mogao Caves, and Palmyra. "
             "Modern Belt and Road Initiative draws geopolitical parallels to the historical Silk Road."),
            ("The Scientific Revolution of the 17th Century",
             "The Scientific Revolution (approx. 1543-1687) transformed how humans understood the natural world, replacing Aristotelian cosmology with mechanistic philosophy. "
             "Key figures: Copernicus (heliocentric model, 1543), Kepler (laws of planetary motion, 1609), Galileo (telescopic astronomy, 1610), "
             "and Newton (universal gravitation, 1687). "
             "Francis Bacon codified the scientific method: systematic observation, hypothesis formation, and experimental verification. "
             "Descartes developed analytical geometry and mind-body dualism, influencing rationalist philosophy. "
             "The Royal Society (1660, London) and Académie des Sciences (1666, Paris) institutionalized scientific collaboration. "
             "This era established science as a distinct intellectual enterprise with its own institutions, methods, and standards of evidence. "
             "The invention of the printing press (Gutenberg, 1440) was an enabling technology, allowing ideas to spread rapidly."),
            ("World War I: Causes and Consequences",
             "World War I (1914-1918) resulted from a tangled web of alliances, militarism, imperialism, and nationalism. "
             "Immediate trigger: the assassination of Archduke Franz Ferdinand of Austria in Sarajevo, June 28, 1914. "
             "The alliance system then drew major powers into war: Triple Entente (Britain, France, Russia) vs. Triple Alliance (Germany, Austria-Hungary, Italy). "
             "Casualties: approximately 20 million dead (including 10 million military) and 21 million wounded. "
             "Key battles — Somme (1916), Verdun (1916), Tannenberg (1914) — exemplified trench warfare's attritional nature. "
             "Weapons of mass destruction included chemical weapons (mustard gas, chlorine), machine guns, and indirect fire artillery. "
             "Treaty of Versailles (1919) imposed crushing reparations on Germany, creating economic devastation and resentment that contributed to WWII. "
             "The war reshaped borders (Ottoman, Austro-Hungarian, and Russian empires collapsed) and triggered the Russian Revolution of 1917."),
            ("The Renaissance: Rebirth of Classical Ideas",
             "The Renaissance ('rebirth') was a cultural, artistic, and intellectual movement originating in Florence, Italy, roughly 1350-1600. "
             "It was driven by: wealth from Mediterranean trade, patronage of arts by the Medici and other merchant families, "
             "the rediscovery of classical Greek and Roman texts, and humanism — the study of human potential and secular concerns. "
             "Key figures include: Leonardo da Vinci (artist, engineer, scientist), Michelangelo (sculptor, painter), "
             "Machiavelli (political philosopher), Brunelleschi (architect), and Gutenberg (printing press). "
             "The Renaissance spread from Italy to Northern Europe (Albrecht Dürer, Erasmus, Shakespeare). "
             "It laid foundations for the Scientific Revolution, Protestant Reformation, and Enlightenment. "
             "Key innovations: linear perspective in art, double-entry bookkeeping, vernacular literature, and humanist education."),
        ],
        "sources": ["wikipedia", "britannica", "national geographic", "history.com", "atlasobscura"],
    },
}

ARTICLE_TEMPLATES = [
    "This article explores the fundamental principles of {topic}, examining both historical context and modern applications. "
    "{body}",
    "An in-depth look at {topic}, from its origins to current state-of-the-art practice. "
    "{body}",
    "A comprehensive introduction to {topic}. This subject has broad implications for both specialists and general readers. "
    "{body}",
    "Understanding {topic} requires examining multiple dimensions: theory, evidence, and real-world practice. "
    "{body}",
    "{topic} is a subject that rewards careful study. This article provides a thorough grounding in core concepts and contemporary debates. "
    "{body}",
]

RECORDS_PER_CATEGORY = 2000  # 10,000 total / 5 categories
RECORDS_PER_TOPIC = RECORDS_PER_CATEGORY // 5  # 400 (5 topics per category)
TOPICS_PER_CATEGORY = 10      # 200 records per topic (informational)

print("Generating 10,000 article records...")

records = []
record_idx = 0

for category, data in CATEGORIES.items():
    topics = data["topics"]
    sources = data["sources"]
    for topic_idx, (title, body_template) in enumerate(topics):
        for sub_idx in range(RECORDS_PER_TOPIC):
            record_id = f"art_{record_idx:04d}"
            body = body_template + (
                f" Additional detail {sub_idx + 1}: "
                f"This article on {title} contributes to the {category} category of our knowledge base. "
                f"Record index {record_idx} of 10,000. Published via {sources[sub_idx % len(sources)]}."
            )
            # Add some seeded variation in length
            if sub_idx % 3 == 0:
                body += (
                    f" Supplementary note: The {category}/{title} topic has been indexed for retrieval. "
                    f"Category code: {category[:4].upper()}, Topic index: {topic_idx}."
                )
            records.append({
                "id": record_id,
                "title": title,
                "body": body,
                "category": category,
                "source": sources[random.randint(0, len(sources) - 1)],
            })
            record_idx += 1

assert record_idx == 10000, f"Expected 10000 records, got {record_idx}"

df = pd.DataFrame(records)
output_path = "test_data/articles.parquet"
df.to_parquet(output_path, index=False, engine="pyarrow")
print(f"Written {len(df)} rows to {output_path}")
print(f"Category distribution:\n{df['category'].value_counts().to_string()}")